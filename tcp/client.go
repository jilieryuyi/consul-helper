package tcp

import (
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
	"sync"
	"context"
	"errors"
)

const asyncWriteChanLen = 10000
var NotConnect          = errors.New("not connect")
const (
	statusConnect = 1 << iota
)

type Client struct {
	ctx               context.Context
	buffer            []byte
	bufferSize        int
	conn              *net.TCPConn
	connLock          *sync.Mutex
	statusLock        *sync.Mutex
	status            int
	onMessageCallback []OnClientEventFunc
	asyncWriteChan    chan []byte
	ip                string
	port              int
	coder             ICoder
	onConnect         OnConnectFunc
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
func SetCoder(coder ICoder) ClientOption {
	return func(tcp *Client) {
		tcp.coder = coder
	}
}

func SetBufferSize(size int) ClientOption {
	return func(tcp *Client) {
		tcp.bufferSize = size
	}
}

func SetOnConnect(onCnnect OnConnectFunc) ClientOption {
	return func(tcp *Client) {
		tcp.onConnect = onCnnect
	}
}

func NewClient(ctx context.Context, ip string, port int, opts ...ClientOption) *Client {
	c := &Client{
		buffer:            make([]byte, 0),
		conn:              nil,
		statusLock:        new(sync.Mutex),
		status:            0,
		onMessageCallback: make([]OnClientEventFunc, 0),
		asyncWriteChan:    make(chan []byte, asyncWriteChanLen),
		connLock:          new(sync.Mutex),
		ip:                ip,
		port:              port,
		ctx:               ctx,
		coder:             &Coder{},
		bufferSize:        4096,
	}
	for _, f := range opts {
		f(c)
	}
	go c.keep()
	return c
}

func (tcp *Client) SetIp(ip string) {
	tcp.ip = ip
}

func (tcp *Client) SetPort(port int) {
	tcp.port = port
}

func (tcp *Client) AsyncWrite(data []byte) {
	tcp.asyncWriteChan <- data
}

func (tcp *Client) Write(data []byte) (int, error) {
	if tcp.status & statusConnect <= 0 {
		return 0, NotConnect
	}
	return tcp.conn.Write(data)
}

func (tcp *Client) keep() {
	data  := tcp.coder.Encode([]byte(""))
	c     := make(chan struct{})
	go func() {
		for {
			c <- struct{}{}
			time.Sleep(time.Second * 3)
		}
	}()
	for {
		select {
		case <- c :
			// keepalive
			tcp.Write(data)
		case sendData, ok := <- tcp.asyncWriteChan:
			//async send support
			if !ok {
				return
			}
			n, err := tcp.Write(sendData)
			if err != nil {
				log.Errorf("send failure: %+v", err)
			}
			if n < len(sendData) {
				log.Errorf("send not complete")
			}
		}
	}
}

// use like go tcp.Connect()
func (tcp *Client) Connect() {
	// 如果已经连接，直接返回
	if tcp.status & statusConnect > 0 {
		return
	}
	for {
		select {
			case <-tcp.ctx.Done():
				return
			default:
		}
		// 断开已有的连接
		tcp.Disconnect()
		// 尝试连接
		for {
			tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", tcp.ip, tcp.port))
			if err != nil {
				log.Errorf("start agent with error: %+v", err)
				break
			}
			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				log.Errorf("start agent with error: %+v", err)
				break
			}
			if tcp.status & statusConnect <= 0 {
				tcp.status |= statusConnect
			}
			tcp.conn = conn
			break
		}
		// 判断连接是否成功
		if tcp.status & statusConnect <= 0 {
			log.Warnf("can not connect to %v:%v, wait a second, will try again", tcp.ip, tcp.port)
			time.Sleep(time.Second)
			continue
		}

		if tcp.onConnect != nil {
			tcp.onConnect(tcp)
		}

		log.Infof("====================client connect to %v:%v ok====================", tcp.ip, tcp.port)
		for {
			if tcp.status & statusConnect <= 0  {
				break
			}
			log.Infof("start read message %v", tcp.bufferSize)
			readBuffer := make([]byte, tcp.bufferSize)
			size, err  := tcp.conn.Read(readBuffer)
			if err != nil || size <= 0 {
				log.Warnf("client read with error: %+v", err)
				tcp.Disconnect()
				break
			}
			fmt.Println("receive: ", string(readBuffer[:size]))
			tcp.onMessage(readBuffer[:size])
			select {
			case <-tcp.ctx.Done():
				return
			default:
			}
		}
	}
}

func (tcp *Client) onMessage(msg []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("onMessage recover%+v, %+v", err, tcp.buffer)
			tcp.buffer = make([]byte, 0)
		}
	}()
	tcp.buffer = append(tcp.buffer, msg...)
	for {
		olen := len(tcp.buffer)
		content, pos, err := tcp.coder.Decode(tcp.buffer)
		if err != nil {
			log.Errorf("%v", err)
			tcp.buffer = make([]byte, 0)
			return
		}
		if len(tcp.buffer) >= pos {
			tcp.buffer = append(tcp.buffer[:0], tcp.buffer[pos:]...)
		} else {
			tcp.buffer = make([]byte, 0)
			log.Errorf("pos %v (olen=%v) error, content=%v(%v) len is %v, data is: %+v", pos, olen, content, string(content), len(tcp.buffer), tcp.buffer)
		}
		for _, f := range tcp.onMessageCallback {
			f(tcp, content)
		}
	}
}

func (tcp *Client) Disconnect() {
	if tcp.status & statusConnect <= 0 {
		log.Debugf("client is in disconnect status")
		return
	}
	log.Warnf("====================client disconnect====================")
	tcp.conn.Close()
	if tcp.status & statusConnect > 0 {
		tcp.status ^= statusConnect
	}
}

