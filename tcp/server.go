package tcp

import (
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
	"context"
)
type TcpClients []*TcpClientNode
type TcpService struct {
	Address string               // 监听ip
	lock *sync.Mutex
	statusLock *sync.Mutex
	listener *net.Listener
	wg *sync.WaitGroup
	agents TcpClients
	status int
	conn *net.TCPConn
	buffer []byte
	ctx context.Context
	index int64
	onMessageCallback []OnServerEventFunc
	codec ICodec
}
type OnServerEventFunc func(node *TcpClientNode, msgId int64, data []byte)
type AgentServerOption func(s *TcpService)

func SetOnServerMessage(f ...OnServerEventFunc) AgentServerOption {
	return func(s *TcpService) {
		s.onMessageCallback = append(s.onMessageCallback, f...)
	}
}

func NewAgentServer(ctx context.Context, address string, opts ...AgentServerOption) *TcpService {
	tcp := &TcpService{
		ctx:              ctx,
		Address:          address,
		lock:             new(sync.Mutex),
		statusLock:       new(sync.Mutex),
		wg:               new(sync.WaitGroup),
		listener:         nil,
		agents:           nil,
		status:           0,
		buffer:           make([]byte, 0),
		index:            0,
		onMessageCallback:   make([]OnServerEventFunc, 0),
		codec:            &Codec{},
	}
	go tcp.keepalive()
	for _, f := range opts {
		f(tcp)
	}
	return tcp
}

func (tcp *TcpService) Start() {
	go func() {
		listen, err := net.Listen("tcp", tcp.Address)
		if err != nil {
			log.Errorf("tcp service listen with error: %+v", err)
			return
		}
		tcp.listener = &listen
		log.Infof("agent service start with: %s", tcp.Address)
		for {
			conn, err := listen.Accept()
			select {
			case <-tcp.ctx.Done():
				return
			default:
			}
			if err != nil {
				log.Warnf("tcp service accept with error: %+v", err)
				continue
			}
			node := newNode(
					tcp.ctx,
					&conn,
					tcp.codec,
					NodeClose(func(n *TcpClientNode) {
						tcp.lock.Lock()
						tcp.agents.remove(n)
						tcp.lock.Unlock()
					}),
				    setOnServerEvents(tcp.onMessageCallback...),
				)
			tcp.lock.Lock()
			tcp.agents.append(node)
			tcp.lock.Unlock()
			go node.readMessage()
		}
	}()
}

func (tcp *TcpService) Broadcast(data []byte) {
	l := int64(len(tcp.agents))
	if l <= 0 {
		return
	}
	for _, client := range tcp.agents {
		client.AsyncSend(data)
	}
}

func (tcp *TcpService) Close() {
	log.Debugf("tcp service closing, waiting for buffer send complete.")
	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	tcp.agents.close()
	log.Debugf("tcp service closed.")
}

// 心跳
func (tcp *TcpService) keepalive() {
	keepalive := tcp.codec.Encode(1, []byte("keepalive response ok"))
	for {
		select {
		case <-tcp.ctx.Done():
			return
		default:
		}
		if tcp.agents == nil {
			time.Sleep(time.Second * 3)
			continue
		}
		tcp.agents.asyncSend(keepalive)
		time.Sleep(time.Second * 3)
	}
}
