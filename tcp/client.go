package tcp

import (
	"net"
	log "github.com/sirupsen/logrus"
	"time"
	"sync"
	"context"
	"errors"
	"sync/atomic"
	"bytes"
)

var (
	NotConnect   = errors.New("not connect")
 	IsConnected  = errors.New("is connected")
 	WaitTimeout  = errors.New("wait timeout")
 	ChanIsClosed = errors.New("wait is closed")
 	UnknownError = errors.New("unknown error")
 	globalMsgId int64 = 1
)
const (
	statusConnect     = 1 << iota
	MaxInt64          = int64(1) << 62
	asyncWriteChanLen = 10000
)
type connectInfo struct {
	address string
	timeout time.Duration
}
type Client struct {
	ctx                 context.Context
	buffer              []byte
	bufferSize          int
	conn                net.Conn
	status              int
	onMessageCallback   []OnClientEventFunc
	asyncWriteChan      chan []byte
	coder               ICodec
	waiter              map[int64] *waiter
	waiterLock          *sync.RWMutex
	waiterGlobalTimeout int64 //毫秒
	wg                  *sync.WaitGroup
	wgAsyncSend         *sync.WaitGroup
	closeChan           chan struct{}
	connectChan         chan *connectInfo
	disConnectChan      chan struct{}
	address string
	connectTimeout time.Duration
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

// 用来设置编码解码的接口
func SetCoder(coder ICodec) ClientOption {
	return func(tcp *Client) {
		tcp.coder = coder
	}
}

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
func SetWaiterGlobalTimeout(timeout int64) ClientOption {
	return func(tcp *Client) {
		tcp.waiterGlobalTimeout = timeout
	}
}

func NewClient(ctx context.Context, address string, opts ...ClientOption) *Client {
	c := &Client{
		buffer:            make([]byte, 0),
		conn:              nil,
		status:            0,
		onMessageCallback: make([]OnClientEventFunc, 0),
		asyncWriteChan:    make(chan []byte, asyncWriteChanLen),
		ctx:               ctx,
		coder:             &Codec{},
		bufferSize:        4096,
		waiter:            make(map[int64]*waiter),
		waiterLock:        new(sync.RWMutex),
		waiterGlobalTimeout: 6000,
		wg:                new(sync.WaitGroup),
		wgAsyncSend:       new(sync.WaitGroup),
		closeChan:         make(chan struct{}),
		connectChan:       make(chan *connectInfo),
		disConnectChan:    make(chan struct{}),
		address: address,
	}
	for _, f := range opts {
		f(c)
	}
	c.connect()
	go c.keepalive()
	go c.asyncWriteProcess()
	go c.keep()
	go c.readMessage()
	return c
}

func (tcp *Client) delWaiter(msgId int64) {
	tcp.waiterLock.Lock()
	w, ok := tcp.waiter[msgId]
	if ok {
		close(w.data)
		delete(tcp.waiter, msgId)
		tcp.wg.Done()
	}
	tcp.waiterLock.Unlock()
}

func (tcp *Client) AsyncSend(data []byte) {
	tcp.wgAsyncSend.Add(1)
	tcp.asyncWriteChan <- data
}

func (tcp *Client) Send(data []byte) (*waiter, int, error) {
	if tcp.status & statusConnect <= 0 {
		return nil, 0, NotConnect
	}
	msgId := atomic.AddInt64(&globalMsgId, 1)
	// check max msgId
	if msgId > MaxInt64 {
		atomic.StoreInt64(&globalMsgId, 1)
		msgId = atomic.AddInt64(&globalMsgId, 1)
	}
	wai := newWaiter(msgId, tcp.delWaiter)
	log.Infof("client.go Client::Send, add waiter, msgId=[%v]", wai.msgId)
	tcp.waiterLock.Lock()
	tcp.waiter[wai.msgId] = wai
	tcp.waiterLock.Unlock()
	tcp.wg.Add(1)
	sendMsg := tcp.coder.Encode(msgId, data)
	num, err  := tcp.conn.Write(sendMsg)
	return wai, num, err
}

// write api 与 send api的差别在于 send 支持同步wait等待服务端响应
// write 则不支持
func (tcp *Client) Write(data []byte) (int, error) {
	if tcp.status & statusConnect <= 0 {
		return 0, NotConnect
	}
	msgId   := atomic.AddInt64(&globalMsgId, 1)
	// check max msgId
	if msgId > MaxInt64 {
		atomic.StoreInt64(&globalMsgId, 1)
		msgId = atomic.AddInt64(&globalMsgId, 1)
	}
	sendMsg := tcp.coder.Encode(msgId, data)
	num, err  := tcp.conn.Write(sendMsg)
	return num, err
}

func (tcp *Client) keepalive() {
	for {
		tcp.Write(keepalivePackage)
		time.Sleep(time.Second * 3)
	}
}

func (tcp *Client) asyncWriteProcess() {
	for {
		select {
		case sendData, ok := <- tcp.asyncWriteChan:
			//async send support
			if !ok {
				return
			}
			tcp.wgAsyncSend.Done()
			_, err := tcp.Write(sendData)
			if err != nil {
				log.Errorf("client.go Client::asyncWriteProcess, send failure: %+v", err)
			}
		}
	}
}

func (tcp *Client) keep() {
	return
	for {
		current := int64(time.Now().UnixNano() / 1000000)
		tcp.waiterLock.Lock()
		for msgId, v := range tcp.waiter  {
			// check timeout
			if current - v.time >= tcp.waiterGlobalTimeout {
				log.Warnf("client.go Client::keep, msgid %v is timeout, will delete", msgId)
				close(v.data)
				delete(tcp.waiter, msgId)
				tcp.wg.Done()
				// 这里为什么不能使用delWaiter的原因是
				// tcp.waiterLock已加锁，而delWaiter内部也加了锁
				// tcp.delWaiter(msgId)
			}
		}
		tcp.waiterLock.Unlock()
		//fmt.Println("#######################tcp.waiter len ", len(tcp.waiter))
		time.Sleep(time.Second * 3)
	}
}

func (tcp *Client) readMessage() {
	for {
		if tcp.status & statusConnect <= 0  {
			tcp.connect()
			time.Sleep(time.Millisecond * 100)
			continue
		}
		readBuffer := make([]byte, tcp.bufferSize)
		size, err  := tcp.conn.Read(readBuffer)
		if isClosedConnError(err) {
			tcp.disconnect()
			continue
		}
		if err != nil || size <= 0 {
			log.Errorf("client.go Client::readMessage, client read with error: %+v", err)
			tcp.disconnect()
			continue
		}
		log.Infof("client.go Client::readMessage, reveive: %v", string(readBuffer[:size]))
		tcp.onMessage(readBuffer[:size])
		select {
		case <-tcp.ctx.Done():
			return
		default:
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
		log.Errorf("client.go Client::Connect, start client with error: %+v", err)
		return err
	}
	tcp.conn = conn
	tcp.status |= statusConnect
	return nil
}

func (tcp *Client) onMessage(msg []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("client.go Client::onMessage, onMessage recover%+v, %+v", err, tcp.buffer)
			tcp.buffer = make([]byte, 0)
		}
	}()
	tcp.buffer = append(tcp.buffer, msg...)
	for {
		bufferLen := len(tcp.buffer)
		msgId, content, pos, err := tcp.coder.Decode(tcp.buffer)
		log.Infof("client.go Client::onMessage, client receive: msgId=%v, data=%v", msgId, string(content))
		if err != nil {
			log.Errorf("%v", err)
			tcp.buffer = make([]byte, 0)
			return
		}
		if msgId <= 0  {
			return
		}
		if len(tcp.buffer) >= pos {
			tcp.buffer = append(tcp.buffer[:0], tcp.buffer[pos:]...)
		} else {
			tcp.buffer = make([]byte, 0)
			log.Errorf("client.go Client::onMessage, pos %v (olen=%v) error, content=%v(%v) len is %v, data is: %+v", pos, bufferLen, content, string(content), len(tcp.buffer), tcp.buffer)
		}
		// 1 is system id
		if msgId > 1 {
			tcp.waiterLock.RLock()
			w, ok := tcp.waiter[msgId]
			tcp.waiterLock.RUnlock()
			data := w.encode(msgId, content)
			if ok {
				log.Infof("client.go Client::onMessage, write waiter: msgId=%v, data=%v", msgId, string(data))
				w.data <- data
			}
		}
		// 判断是否是心跳包，心跳包不触发回调函数
		if !bytes.Equal(keepalivePackage, content) {
			for _, f := range tcp.onMessageCallback {
				f(tcp, content)
			}
		}
	}
}

func (tcp *Client) disconnect() error {
	tcp.wg.Wait()
	tcp.wgAsyncSend.Wait()
	if tcp.status & statusConnect <= 0 {
		return NotConnect
	}
	log.Infof("disconnect was called")
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
	err := tcp.disconnect()
	if err != nil {
		log.Errorf("Close disconnect fail, err=[%v]", err)
	}
	close(tcp.asyncWriteChan)
}


