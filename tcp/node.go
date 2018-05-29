package tcp

import (
	"time"
	"net"
	log "github.com/sirupsen/logrus"
	"sync"
	"io"
	"context"
)

type TcpClientNode struct {
	conn *net.Conn   // 客户端连接进来的资源句柄
	sendQueue chan []byte // 发送channel
	sendFailureTimes int64       // 发送失败次数
	recvBuf []byte      // 读缓冲区
	connectTime int64       // 连接成功的时间戳
	recvBufLock *sync.Mutex
	status int
	wg *sync.WaitGroup
	lock *sync.Mutex          // 互斥锁，修改资源时锁定
	onclose []NodeFunc
	ctx context.Context
	onServerEvents []OnServerEventFunc
}

type OnPullCommandFunc func(node *TcpClientNode)

func setOnServerEvents(f ...OnServerEventFunc) NodeOption {
	return func(n *TcpClientNode) {
		n.onServerEvents = append(n.onServerEvents, f...)
	}
}

func newNode(ctx context.Context, conn *net.Conn, opts ...NodeOption) *TcpClientNode {
	node := &TcpClientNode{
		conn:             conn,
		sendQueue:        make(chan []byte, tcpMaxSendQueue),
		sendFailureTimes: 0,
		connectTime:      time.Now().Unix(),
		recvBuf:          make([]byte, 0),
		status:           tcpNodeOnline,
		ctx:              ctx,
		lock:             new(sync.Mutex),
		onclose:          make([]NodeFunc, 0),
		wg:               new(sync.WaitGroup),
		onServerEvents:   make([]OnServerEventFunc, 0),
		recvBufLock:      new(sync.Mutex),
	}
	for _, f := range opts {
		f(node)
	}
	go node.asyncSendService()
	return node
}

func NodeClose(f NodeFunc) NodeOption {
	return func(n *TcpClientNode) {
		n.onclose = append(n.onclose, f)
	}
}

func (node *TcpClientNode) close() {
	node.lock.Lock()
	if node.status & tcpNodeOnline <= 0 {
		node.lock.Unlock()
		return
	}
	if node.status & tcpNodeOnline > 0{
		node.status ^= tcpNodeOnline
		(*node.conn).Close()
		close(node.sendQueue)
	}
	log.Warnf("node close")
	node.lock.Unlock()
	for _, f := range node.onclose {
		f(node)
	}
}

func (node *TcpClientNode) Send(data []byte) (int, error) {
	(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 3))
	return (*node.conn).Write(data)
}

func (node *TcpClientNode) AsyncSend(data []byte) {
	node.lock.Lock()
	if node.status & tcpNodeOnline <= 0 {
		node.lock.Unlock()
		return
	}

	//start1 := time.Now()
	for {
		if len(node.sendQueue) < cap(node.sendQueue) {
			break
		}
		log.Warnf("cache full, try wait, %v, %v", len(node.sendQueue) , cap(node.sendQueue))
	}
	node.sendQueue <- data
	node.lock.Unlock()
	//log.Debugf("############AsyncSend use time========= %+v", time.Since(start1))
}

//func (node *TcpClientNode) SendKeep(data []byte) {
//	node.keepLock.Lock()
//	node.keep[]
//	node.keepLock.Unlock()
//}

func (node *TcpClientNode) setReadDeadline(t time.Time) {
	(*node.conn).SetReadDeadline(t)
}

func (node *TcpClientNode) asyncSendService() {
	node.wg.Add(1)
	defer node.wg.Done()
	for {
		if node.status & tcpNodeOnline <= 0 {
			log.Info("tcp node is closed, clientSendService exit.")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			//start := time.Now()
			if !ok {
				log.Info("tcp node sendQueue is closed, sendQueue channel closed.")
				return
			}
			//if len(msg) == 1 && msg[0] == byte(0) {
			//	close(node.sendQueue)
			//	log.Warnf("close sendQueue")
			//	return
			//}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 30))
			size, err := (*node.conn).Write(msg)
			//log.Debugf("send: %+v, to %+v", msg, (*node.conn).RemoteAddr().String())
			if err != nil {
				log.Errorf("tcp send to %s error: %v", (*node.conn).RemoteAddr().String(), err)
				node.close()
				return
			}
			if size != len(msg) {
				log.Errorf("%s send not complete: %v", (*node.conn).RemoteAddr().String(), msg)
			}
			//fmt.Fprintf(os.Stderr, "write use time %v\r\n", time.Since(start))
		case <-node.ctx.Done():
			log.Debugf("context is closed, wait for exit, left: %d", len(node.sendQueue))
			if len(node.sendQueue) <= 0 {
				log.Info("tcp service, clientSendService exit.")
				return
			}
		}
	}
}

func (node *TcpClientNode) onMessage(msg []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Unpack recover##########%+v, %+v", err, node.recvBuf)
			node.recvBuf = make([]byte, 0)
		}
	}()
	node.recvBuf = append(node.recvBuf, msg...)
	for {
		olen := len(node.recvBuf)
		cmd, content, pos, err := Unpack(node.recvBuf)
		if err != nil {
			node.recvBuf = make([]byte, 0)
			log.Errorf("node.recvBuf error %v", err)
			return
		}
		if cmd <= 0 {
			return
		}
		if len(node.recvBuf) >= pos {
			node.recvBuf = append(node.recvBuf[:0], node.recvBuf[pos:]...)
		} else {
			node.recvBuf = make([]byte, 0)
			log.Errorf("pos %v(olen=%v) error, cmd=%v, content=%v(%v) len is %v, data is: %+v", pos,olen, cmd, content, string(content), len(node.recvBuf), node.recvBuf)
			//return
		}
		if !hasCmd(cmd) {
			node.recvBuf = make([]byte, 0)
			log.Errorf("cmd（%v）does not exists", cmd)
			return
		}
		for _, f := range node.onServerEvents {
			f(node, cmd, content)
		}
	}
}

//func (node *TcpClientNode) eventFired(event int, data []byte) {
//	for _, f := range node.onevents {
//		f(event, data)
//	}
//}

func (node *TcpClientNode) readMessage() {
	//node := newNode(tcp.ctx, conn, NodeClose(tcp.agents.remove), NodePro(tcp.agents.append))

	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	for {
		readBuffer := make([]byte, 4096)
		size, err := (*node.conn).Read(readBuffer)
		if err != nil {
			if err != io.EOF {
				log.Warnf("tcp node %s disconnect with error: %v", (*node.conn).RemoteAddr().String(), err)
			} else {
				log.Debugf("tcp node %s disconnect with error: %v", (*node.conn).RemoteAddr().String(), err)
			}
			node.close()
			return
		}
		//log.Debugf("#####################server receive message: %+v", readBuffer[:size])
		node.onMessage(readBuffer[:size])
	}
}


