package tcp

import (
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
	"context"
)

type Server struct {
	Address           string
	lock              *sync.Mutex
	statusLock        *sync.Mutex
	listener          *net.Listener
	clients           Clients
	status            int
	conn              *net.TCPConn
	buffer            []byte
	ctx               context.Context
	onMessageCallback []OnServerMessageFunc
	codec             ICodec
	cancel            context.CancelFunc
}
type Clients             []*ClientNode
type OnServerMessageFunc func(node *ClientNode, msgId int64, data []byte)
type ServerOption        func(s *Server)
var keepalivePackage     = []byte{
	byte(0),byte(0), byte(0), byte(0),
	byte('k'), byte('e'), byte('e'), byte('p'),
	byte('a'),byte('l'),byte('i'),byte('v'),byte('e'),
	byte(0),byte(0), byte(0), byte(0),
}

// set receive msg callback func
func SetOnServerMessage(f ...OnServerMessageFunc) ServerOption {
	return func(s *Server) {
		s.onMessageCallback = append(s.onMessageCallback, f...)
	}
}

// set codec, codes use for encode and descode msg
// codec must implement from ICodec
func SetServerCodec(codec ICodec) ServerOption {
	return func(s *Server) {
		s.codec = codec
	}
}

// new a tcp server
// ctx like content.Background
// address like 127.0.0.1:7770
// opts like
// tcp.SetOnServerMessage(func(node *tcp.ClientNode, msgId int64, data []byte) {
//		node.Send(msgId, data)
// })
func NewServer(c context.Context, address string, opts ...ServerOption) *Server {
	ctx, cancel := context.WithCancel(c)
		tcp := &Server{
		ctx:               ctx,
		Address:           address,
		lock:              new(sync.Mutex),
		statusLock:        new(sync.Mutex),
		listener:          nil,
		clients:           make(Clients, 0),
		status:            0,
		buffer:            make([]byte, 0),
		onMessageCallback: make([]OnServerMessageFunc, 0),
		codec:             &Codec{},
		cancel:            cancel,
	}
	go tcp.keepalive()
	for _, f := range opts {
		f(tcp)
	}
	return tcp
}

// start tcp service
func (tcp *Server) Start() {
	go func(){
		listen, err := net.Listen("tcp", tcp.Address)
		if err != nil {
			log.Panicf("server.go Server::Start, tcp service listen with error=[%+v]", err)
			return
		}
		tcp.listener = &listen
		log.Infof("server.go Server::Start, tcp service start with: %s", tcp.Address)
		for {
			select {
			case <-tcp.ctx.Done():
				return
			default:
			}
			conn, err := listen.Accept()
			if isClosedConnError(err) {
				return
			}
			if err != nil {
				log.Errorf("server.go Server::Start, accept error, err=[%+v]", err)
				continue
			}
			node := newNode(
				tcp.ctx,
				&conn,
				tcp.codec,
				setOnNodeClose(func(n *ClientNode) {
					tcp.lock.Lock()
					tcp.clients.remove(n)
					tcp.lock.Unlock()
				}),
				setOnMessage(tcp.onMessageCallback...),
			)
			tcp.lock.Lock()
			tcp.clients.append(node)
			tcp.lock.Unlock()
			go node.readMessage()
		}
	}()
}

// Broadcast data to all connected clients
func (tcp *Server) Broadcast(msgId int64, data []byte) {
	for _, client := range tcp.clients {
		client.AsyncSend(msgId, data)
	}
}

// close service
func (tcp *Server) Close() {
	tcp.cancel()
	log.Infof("server.go Server::Close, tcp service closing, waiting for buffer send complete.")
	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	tcp.clients.close()
	log.Infof("server.go Server::Close done.")
}

// keepalive
func (tcp *Server) keepalive() {
	for {
		select {
		case <-tcp.ctx.Done():
			return
		default:
		}
		tcp.clients.asyncSend(1, keepalivePackage)
		time.Sleep(time.Second * 3)
	}
}
