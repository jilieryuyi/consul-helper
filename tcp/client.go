package agent

import (
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
	"sync"
	"context"
	"errors"
)

type Client struct {
	ctx            context.Context
	buffer         []byte
	conn           *net.TCPConn
	connLock       *sync.Mutex
	statusLock     *sync.Mutex
	status         int
	getLeader      GetLeaderFunc
	onEvents       []OnClientEventFunc
	asyncWriteChan chan []byte
	checkChan      chan address
}
type address struct {
	ip string
	port int
}
type GetLeaderFunc     func()(string, int, error)
type ClientOption      func(tcp *Client)
type OnClientEventFunc func(tcp *Client, event int, content []byte)

const asyncWriteChanLen = 10000
var notConnect          = errors.New("not connect")

func SetGetLeader(f GetLeaderFunc) ClientOption {
	return func(tcp *Client) {
		tcp.getLeader = f
	}
}

func SetOnClientEvent(f ...OnClientEventFunc) ClientOption {
	return func(tcp *Client) {
		tcp.onEvents = append(tcp.onEvents, f...)
	}
}

func NewClient(ctx context.Context, opts ...ClientOption) *Client {
	c := &Client{
		ctx:           ctx,
		buffer:        make([]byte, 0),
		conn:          nil,
		statusLock:    new(sync.Mutex),
		status:        0,
		onEvents:      make([]OnClientEventFunc, 0),
		asyncWriteChan:make(chan []byte, asyncWriteChanLen),
		connLock:      new(sync.Mutex),
		checkChan:     make(chan address),
	}
	for _, f := range opts {
		f(c)
	}
	go c.keep()
	return c
}

func (tcp *Client) AsyncWrite(data []byte) {
	tcp.asyncWriteChan <- data
}

func (tcp *Client) Write(data []byte) (int, error) {
	if tcp.status & agentStatusConnect <= 0 {
		return 0, notConnect
	}
	return tcp.conn.Write(data)
}

func (tcp *Client) connect(ip string, port int) {
	if ip == "" || port <= 0 {
		return
	}
	tcp.disconnect()
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		log.Errorf("start agent with error: %+v", err)
		return
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Errorf("start agent with error: %+v", err)
		return
	}
	if tcp.status & agentStatusConnect <= 0 {
		tcp.status |= agentStatusConnect
	}
	tcp.conn = conn
}

func (tcp *Client) keep() {
	data         := Pack(CMD_TICK, []byte(""))
	var c         = make(chan struct{})
	var serviceIp = ""
	var port      = 0

	go func() {
		for {
			c <- struct{}{}
			time.Sleep(time.Second * 3)
		}
	}()

	for {
		select {
		case ad, ok := <- tcp.checkChan:
			if !ok {
				return
			}
			if serviceIp != "" && port > 0 {
				if serviceIp != ad.ip || port != ad.port {
					log.Warnf("leader change found")
					//如果服务地址端口发生改变
					tcp.disconnect()
					serviceIp = ad.ip
					port = ad.port
				}
			} else {
				serviceIp = ad.ip
				port = ad.port
				tcp.run()
			}
		case <- c :
			s, p, _ := tcp.getLeader()
			if serviceIp != "" && port > 0 {
				if serviceIp != s || port != p {
					log.Warnf("self check, leader change found")
					//如果服务地址端口发生改变
					tcp.disconnect()
					serviceIp = s//ad.ip
					port = p//ad.port
				}
			}
			tcp.Write(data)
		case sendData, ok := <- tcp.asyncWriteChan:
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

func (tcp *Client) Start(serviceIp string, port int)  {
	tcp.checkChan <- address{serviceIp, port}
}

func (tcp *Client) run() {
	if tcp.status & agentStatusConnect > 0 {
		return
	}
	go func() {
		for {
			select {
				case <-tcp.ctx.Done():
					return
				default:
			}
			serviceIp, port, _ := tcp.getLeader()
			tcp.connect(serviceIp, port)
			if tcp.status & agentStatusConnect <= 0 {
				time.Sleep(time.Second * 3)
				for {
					serviceIp, port, _ = tcp.getLeader()
					if serviceIp == "" || port <= 0 {
						log.Warnf("ip or port empty: %v, %v, wait for init", serviceIp, port)
						time.Sleep(time.Second * 1)
						continue
					}
					break
				}
				continue
			}
			log.Debugf("====================agent client connect to leader %s:%d====================", serviceIp, port)
			for {
				if tcp.status & agentStatusConnect <= 0  {
					break
				}
				readBuffer := make([]byte, 4096)
				size, err  := tcp.conn.Read(readBuffer)

				if err != nil || size <= 0 {
					log.Warnf("agent read with error: %+v", err)
					tcp.disconnect()
					break
				}
				tcp.onMessage(readBuffer[:size])
				select {
				case <-tcp.ctx.Done():
					return
				default:
				}
			}
		}
	}()
}

func (tcp *Client) onMessage(msg []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Unpack recover##########%+v, %+v", err, tcp.buffer)
			tcp.buffer = make([]byte, 0)
		}
	}()
	tcp.buffer = append(tcp.buffer, msg...)
	for {
		olen := len(tcp.buffer)
		cmd, content, pos, err := Unpack(tcp.buffer)
		if err != nil {
			log.Errorf("%v", err)
			tcp.buffer = make([]byte, 0)
			return
		}
		if cmd <= 0 {
			return
		}
		if len(tcp.buffer) >= pos {
			tcp.buffer = append(tcp.buffer[:0], tcp.buffer[pos:]...)
		} else {
			tcp.buffer = make([]byte, 0)
			log.Errorf("pos %v (olen=%v) error, cmd=%v, content=%v(%v) len is %v, data is: %+v", pos, olen, cmd, content, string(content), len(tcp.buffer), tcp.buffer)
		}
		if !hasCmd(cmd) {
			log.Errorf("cmd %d dos not exists", cmd)
			tcp.buffer = make([]byte, 0)
			return
		}
		for _, f := range tcp.onEvents {
			f(tcp, cmd, content)
		}
	}
}

func (tcp *Client) disconnect() {
	if tcp.status & agentStatusConnect <= 0 {
		log.Debugf("agent is in disconnect status")
		return
	}
	log.Warnf("====================agent disconnect====================")
	tcp.conn.Close()
	if tcp.status & agentStatusConnect > 0 {
		tcp.status ^= agentStatusConnect
	}
}

