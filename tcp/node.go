package tcp

import (
	"net"
	log "github.com/sirupsen/logrus"
	"sync"
	"context"
)

const (
	tcpMaxSendQueue = 10000
	tcpNodeOnline   = 1 << iota
)
type NodeFunc   func(n *ClientNode)
type NodeOption func(n *ClientNode)

type ClientNode struct {
	conn              *net.Conn
	sendQueue         chan []byte
	recvBuf           []byte
	status            int
	wg                *sync.WaitGroup
	lock              *sync.Mutex
	onclose           []NodeFunc
	ctx               context.Context
	onMessageCallback []OnServerMessageFunc
	codec             ICodec
}

func setOnMessage(f ...OnServerMessageFunc) NodeOption {
	return func(n *ClientNode) {
		n.onMessageCallback = append(n.onMessageCallback, f...)
	}
}

func setOnNodeClose(f NodeFunc) NodeOption {
	return func(n *ClientNode) {
		n.onclose = append(n.onclose, f)
	}
}

func newNode(ctx context.Context, conn *net.Conn, codec ICodec, opts ...NodeOption) *ClientNode {
	node := &ClientNode{
		conn:              conn,
		sendQueue:         make(chan []byte, tcpMaxSendQueue),
		recvBuf:           make([]byte, 0),
		status:            tcpNodeOnline,
		ctx:               ctx,
		lock:              new(sync.Mutex),
		onclose:           make([]NodeFunc, 0),
		wg:                new(sync.WaitGroup),
		onMessageCallback: make([]OnServerMessageFunc, 0),
		codec:             codec,
	}
	for _, f := range opts {
		f(node)
	}
	go node.asyncSendService()
	return node
}

func (node *ClientNode) close() {
	node.lock.Lock()
	defer node.lock.Unlock()
	if node.status & tcpNodeOnline <= 0 {
		return
	}
	if node.status & tcpNodeOnline > 0{
		node.status ^= tcpNodeOnline
		(*node.conn).Close()
		close(node.sendQueue)
	}
	log.Infof("close, node close")
	for _, f := range node.onclose {
		f(node)
	}
}

func (node *ClientNode) Send(msgId int64, data []byte) (int, error) {
	sendData := node.codec.Encode(msgId, data)
	n, e := (*node.conn).Write(sendData)
	if e != nil {
		return n, e
	}
	if n != len(sendData) {
		return n, SendNotComplate
	}
	return len(data), nil
}

func (node *ClientNode) AsyncSend(msgId int64, data []byte) {
	node.lock.Lock()
	defer node.lock.Unlock()
	if node.status & tcpNodeOnline <= 0 {
		return
	}
	node.sendQueue <- node.codec.Encode(msgId, data)
}

func (node *ClientNode) asyncSendService() {
	node.wg.Add(1)
	defer node.wg.Done()
	for {
		if node.status & tcpNodeOnline <= 0 {
			log.Info("asyncSendService, tcp node is closed, clientSendService exit.")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Info("asyncSendService, tcp node sendQueue is closed, sendQueue channel closed.")
				return
			}
			size, err := (*node.conn).Write(msg)
			if err != nil {
				log.Errorf("asyncSendService, tcp send to %s error: %v", (*node.conn).RemoteAddr().String(), err)
				node.close()
				return
			}
			if size != len(msg) {
				log.Errorf("asyncSendService, %s send not complete: %v", (*node.conn).RemoteAddr().String(), msg)
			}
		case <- node.ctx.Done():
			log.Debugf("asyncSendService, context is closed, wait for exit, left: %d", len(node.sendQueue))
			if len(node.sendQueue) <= 0 {
				log.Info("asyncSendService, tcp service, clientSendService exit.")
				return
			}
		}
	}
}

func (node *ClientNode) readMessage() {
	var frame = newPackage(*node.conn)
	for {
		content, msgId, err := frame.parse()
		if  err != nil {
			log.Errorf("readMessage fail, client host=[%v], err=[%v]", (*node.conn).RemoteAddr().String(), err)
			node.close()
			return
		}
		for _, f := range node.onMessageCallback {
			f(node, msgId, content)
		}
	}
}


