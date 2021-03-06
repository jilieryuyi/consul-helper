package tcp

import (
	"net"
	log "github.com/sirupsen/logrus"
	"time"
	"sync"
	"context"
	"bytes"
)

const (
	statusConnect        = 1 << iota
	MaxInt64             = int64(1) << 62
	asyncWriteChanLen    = 10000
	defaultWaiterTimeout = 6000
	defaultWriteTimeout  = 6//秒
	defaultBufferSize    = 4096
)
type Client struct {
	ctx                 context.Context
	buffer              []byte
	bufferSize          int
	conn                net.Conn
	status              int
	onMessageCallback   []OnClientEventFunc
	asyncWriteChan      chan []byte
	waiterGlobalTimeout int64 //毫秒
	wg                  *sync.WaitGroup
	wgAsyncSend         *sync.WaitGroup
	address             string
	connectTimeout      time.Duration
	cancel              context.CancelFunc
	waiterPool          *waiterPool
	waiterManager       *waiterManager
}

type ClientOption      func(tcp *Client)
type OnClientEventFunc func(tcp *Client, content []byte)
type OnConnectFunc     func(tcp *Client)

// 设置收到消息的回调函数
// 回调函数同步执行，不能使阻塞的函数
func SetOnMessage(f ...OnClientEventFunc) ClientOption {
	return func(tcp *Client) {
		tcp.onMessageCallback = append(tcp.onMessageCallback, f...)
	}
}

// 设置连接超时
func SetClientConnectTimeout(timeout time.Duration) ClientOption {
	return func(tcp *Client) {
		tcp.connectTimeout = timeout
	}
}

// 设置缓冲区大小
func SetBufferSize(size int) ClientOption {
	return func(tcp *Client) {
		tcp.bufferSize = size
	}
}

// 单位是毫秒
// 设置waiter检测的超时时间，默认为6000毫秒
// 如果超过该时间，waiter就会被删除
func SetWaiterTimeout(timeout int64) ClientOption {
	return func(tcp *Client) {
		tcp.waiterGlobalTimeout = timeout
	}
}

// 创建一个tcp客户端
// 第一个参数为context上下文
// 第二个参数是客户端将要连接的目标地址，如： 127.0.0.1:9998
// 后面的参数为可选参数
// 返回值为客户端对象和err错误信息
// 对应的错误为连接目标tcp地址出错时返回
func NewClient(ctx context.Context, address string, opts ...ClientOption) (*Client, error) {
	ctx, cancel := context.WithCancel(ctx)
	c := &Client{
		buffer:              make([]byte, 0),
		conn:                nil,
		status:              0,
		onMessageCallback:   make([]OnClientEventFunc, 0),
		asyncWriteChan:      make(chan []byte, asyncWriteChanLen),
		ctx:                 ctx,
		bufferSize:          defaultBufferSize,
		wg:                  new(sync.WaitGroup),
		wgAsyncSend:         new(sync.WaitGroup),
		address:             address,
		cancel:              cancel,
		waiterPool:          newWaiterPool(1024),
		waiterManager:       newWaiterManager(ctx),
	}
	for _, f := range opts {
		f(c)
	}
	err := c.connect()
	if err != nil {
		close(c.asyncWriteChan)
		cancel()
		return nil, err
	}
	go c.keepalive()
	go c.asyncWriteProcess()
	go c.readMessage()
	return c, nil
}

// 异步发送消息
func (tcp *Client) AsyncSend(data []byte) {
	tcp.wgAsyncSend.Add(1)
	tcp.asyncWriteChan <- data
}

// 同步发送消息，支持同步等效消息响应
// 同步获取响应结果通过waiter的api Wait支持
// 参数为需要发送的消息
// 返回值微分三个
// 第一个返回值为waiter对象
// 第二个返回值为已发送的消息大小
// 最后一个为错误信息，如果没有错误发生，此值为nil
// 连接已断开时返回NotConnect
// 其他错误值为tcp发送错误的返回值
func (tcp *Client) Send(data []byte, writeTimeout time.Duration) (*waiter, int, error) {
	log.Infof("Client::Send, msg=[%v, %+v]", string(data), data)
	if tcp.status & statusConnect <= 0 {
		log.Errorf("Client::Send fail, msg=[%v, %+v], err=[%v]", string(data), data, NotConnect)
		return nil, 0, NotConnect
	}
	// 获取消息id
	msgId   := getMsgId()
	sendMsg := Encode(msgId, data)

	// 设置写超时时间
	if writeTimeout > 0 {
		err := tcp.conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
		if err != nil {
			log.Errorf("Client::Send SetWriteDeadline fail, msg=[%v, %+v], err=[%v]", string(data), data, err)
			return nil, 0, err
		}
		defer tcp.conn.SetWriteDeadline(time.Time{})
	}

	// 这里有一个坑
	// 必须要在conn.Write之前写入tcp.waiter
	// 如果在之后写入，可能出现服务端收到消息并且响应到client先于waiter写入
	// 这个时候就会出现waiter找不到的情况
	wai, err := tcp.waiterPool.get(msgId, tcp.waiterManager.clear)//newWaiter(msgId, tcp.delWaiter)
	if err != nil {
		return nil, 0, err
	}
	tcp.waiterManager.append(wai)

	// 发送消息
	num, err := tcp.conn.Write(sendMsg)
	if num != len(sendMsg) {
		log.Errorf("Client::Send Write not complete, msg=[%v, %+v]", string(data), data)
	}

	// 如果成功（没有错误发生）
	// 则返回waiter支持
	if err == nil {
		log.Infof("Client::Send success add waiter, msgId=[%v], msg=[%v, %+v]", msgId, string(data), data)
		return wai, num, nil
	}

	log.Infof("Send clear %v", msgId)
	// 如果发生了错误，不用返回waiter
	// 这个时候直接关掉waiter
	tcp.waiterManager.clear(msgId)
	//tcp.waiterLock.Lock()
	//wai.StopWait()
	//wai.msgId = 0
	//wai.onComplete = nil
	//delete(tcp.waiter, msgId)
	//tcp.waiterLock.Unlock()
	// 发生错误
	log.Errorf("Client::Send Write fail, msg=[%v, %+v], err=[%v]", string(data), data, err)
	return nil, num, err
}

// write api 与 send api的差别在于 send 支持同步wait等待服务端响应
// write 则不支持
// 返回值为发送消息大小和发生的错误
func (tcp *Client) Write(data []byte, writeTimeout time.Duration) (int, error) {
	if tcp.status & statusConnect <= 0 {
		return 0, NotConnect
	}
	// 设置写超时时间
	if writeTimeout > 0 {
		err := tcp.conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
		if err != nil {
			log.Errorf("Client::Write SetWriteDeadline fail, msg=[%v, %+v], err=[%v]", string(data), data, err)
			return 0, err
		}
		defer tcp.conn.SetWriteDeadline(time.Time{})
	}

	msgId   := getMsgId()
	sendMsg := Encode(msgId, data)
	num, err  := tcp.conn.Write(sendMsg)
	// 判断消息是否发生完整
	if num != len(sendMsg) {
		log.Errorf("Client::Write Write not complete, msg=[%v, %+v]", string(data), data)
	}
	// 无错误发生
	if err == nil {
		return num, err
	}
	log.Errorf("Client::Write Write fail, msg=[%v, %+v], err=[%v]", string(data), data, err)
	return num, nil
}

func (tcp *Client) keepalive() {
	for {
		select {
		case <-tcp.ctx.Done():
			return
		default:
		}
		if tcp.conn == nil {
			time.Sleep(time.Second * 3)
			continue
		}
		if tcp.status & statusConnect <= 0  {
			time.Sleep(time.Second * 3)
			continue
		}
		tcp.conn.Write(Encode(1, keepalivePackage))
		//tcp.Write(keepalivePackage)
		time.Sleep(time.Second * 3)
	}
}

func (tcp *Client) asyncWriteProcess() {
	for {
		select {
		case sendData, ok := <- tcp.asyncWriteChan:
			if !ok {
				return
			}
			tcp.wgAsyncSend.Done()
			_, err := tcp.Write(sendData, time.Second * defaultWriteTimeout)
			if err != nil {
				log.Errorf("Client::asyncWriteProcess Write fail, err=[%+v]", err)
			}
		case <-tcp.ctx.Done():
			if len(tcp.asyncWriteChan) <= 0 {
				return
			}
		}
	}
}

func (tcp *Client) readMessage() {
	//var bio = bufio.NewReader(tcp.conn)
	//var header = make([]byte, 4)
	//var packageLen = make([]byte, 4)
	//var msgIdBuf = make([]byte, 8)
	var frame = newPackage(tcp.conn)
	for {
		select {
			case <-tcp.ctx.Done():
				return
			default:
		}
		// 如果当前状态为离线，尝试重新连接
		if tcp.status & statusConnect <= 0  {
			tcp.connect()
			time.Sleep(time.Millisecond * 100)
			continue
		}

		content, msgId, err := frame.parse()
		if err != nil {
			log.Errorf("readMessage header fail, err=[%v]", err)
			tcp.disconnect()
			continue
		}

		// 1 is system id
		if msgId > 1 {
			tcp.waiterManager.get(msgId).post(msgId, content)
		}
		// 判断是否是心跳包，心跳包不触发回调函数
		if !bytes.Equal(keepalivePackage, content) {
			for _, f := range tcp.onMessageCallback {
				f(tcp, content)
			}
		}
	}
}

// use like go tcp.Connect()
func (tcp *Client) connect() error {
	// 如果已经连接，直接返回
	if tcp.status & statusConnect > 0 {
		return IsConnected
	}
	dial := net.Dialer{Timeout: tcp.connectTimeout}
	conn, err := dial.Dial("tcp", tcp.address)
	if err != nil {
		log.Errorf("Connect Dial fail, err=[%+v]", err)
		return err
	}
	tcp.conn = conn
	tcp.status |= statusConnect
	return nil
}

//func (tcp *Client) onMessage(msg []byte) {
//	defer func() {
//		if err := recover(); err != nil {
//			log.Errorf("Client::onMessage recover, err=[%+v], buffer=[%+v, %+v]", err, string(tcp.buffer), tcp.buffer)
//			tcp.buffer = make([]byte, 0)
//		}
//	}()
//	tcp.buffer = append(tcp.buffer, msg...)
//	for {
//		bufferLen := len(tcp.buffer)
//		msgId, content, pos, err := tcp.coder.Decode(tcp.buffer)
//		log.Infof("client.go Client::onMessage, client receive: msgId=[%v], data=[%v, %v]", msgId, string(content), content)
//		if err != nil {
//			log.Errorf("Client::onMessage coder Decode fail, err=[%v]", err)
//			tcp.buffer = make([]byte, 0)
//			return
//		}
//		if msgId <= 0  {
//			return
//		}
//		if len(tcp.buffer) >= pos {
//			tcp.buffer = append(tcp.buffer[:0], tcp.buffer[pos:]...)
//		} else {
//			tcp.buffer = make([]byte, 0)
//			log.Errorf("Client::onMessage pos error, pos=[%v], bufferLen=[%v], content=[(%v) %v], buffer=[(%+v) %v", pos, bufferLen, string(content), content, len(tcp.buffer), tcp.buffer)
//		}
//		// 1 is system id
//		if msgId > 1 {
//			tcp.waiterManager.get(msgId).post(msgId, content)
//		}
//		// 判断是否是心跳包，心跳包不触发回调函数
//		if !bytes.Equal(keepalivePackage, content) {
//			for _, f := range tcp.onMessageCallback {
//				f(tcp, content)
//			}
//		}
//	}
//}

func (tcp *Client) disconnect() error {
	log.Infof("disconnect was called")
	//tcp.wg.Wait()
	// 等待异步发送全部发送完毕
	tcp.wgAsyncSend.Wait()
	if tcp.status & statusConnect <= 0 {
		log.Errorf("disconnect fail, err=[%v]", NotConnect)
		return NotConnect
	}

	//tcp.waiterLock.Lock()
	//for msgId, v := range tcp.waiter  {
	//	log.Infof("disconnect, %v stop wait", msgId)
	//	v.StopWait()
	//	v.onComplete = nil
	//	v.msgId = 0
	//	//close(v.data)
	//	delete(tcp.waiter, msgId)
	//}
	//tcp.waiterLock.Unlock()
	tcp.waiterManager.clearAll()
	err := tcp.conn.Close()
	if tcp.status & statusConnect > 0 {
		tcp.status ^= statusConnect
	}
	if err != nil {
		log.Errorf("disconnect fail, err=[%v]", err)
	}
	return err
}

func (tcp *Client) Close() {
	tcp.cancel()
	err := tcp.disconnect()
	if err != nil {
		log.Errorf("Close disconnect fail, err=[%v]", err)
	}
	close(tcp.asyncWriteChan)
}


