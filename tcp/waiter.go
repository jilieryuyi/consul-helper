package tcp

import (
	"time"
	"encoding/binary"
	log "github.com/sirupsen/logrus"
)

type waiter struct {
	msgId     int64
	data      chan []byte
	time      int64
	onComplete func(int64)
	client *Client
	exitWait chan struct{}
}


func newWaiter(msgId int64, onComplete func(i int64)) *waiter {
	//if onComplete == nil {
	//	onComplete = func(i int64) {
	//		// just for some debug
	//		fmt.Println("######", i, " is complete######")
	//	}
	//}
	return &waiter{
		msgId: msgId,
		data:  make(chan []byte, 1),
		time:  int64(time.Now().UnixNano() / 1000000),
		onComplete: onComplete,
		exitWait: make(chan struct{}, 3),
	}
}

// 编码等待的数据
func (w *waiter) encode(msgId int64, raw []byte) []byte {
	if w == nil {
		return nil
	}
	data := make([]byte, 8 + len(raw))
	binary.LittleEndian.PutUint64(data[:8], uint64(msgId))
	copy(data[8:], raw)
	return data
}

// 解码等待的数据
func (w *waiter) decode(data []byte) (int64, []byte) {
	if w == nil {
		return 0, nil// nil, 0, WaiterNil
	}
	msgId := int64(binary.LittleEndian.Uint64(data[:8]))
	return msgId, data[8:]
}

// 手动终止等待
func (w *waiter) StopWait() {
	if w == nil {
		return// nil, 0, WaiterNil
	}
	w.exitWait <- struct{}{}
}

// 如果timeout <= 0 永不超时
func (w *waiter) Wait(timeout time.Duration) ([]byte, int64, error) {
	if w == nil {
		return nil, 0, WaiterNil
	}
	// if timeout is 0, never timeout
	if timeout <= 0 {
			select {
			case data, ok := <-w.data:
				if !ok {
					return nil, 0, nil //ChanIsClosed
				}
				msgId, raw := w.decode(data)
				if w.onComplete != nil {
					w.onComplete(msgId)
				}
				return raw, msgId, nil
			case <-w.exitWait:
				log.Infof("get Interrupt sig")
				if w.onComplete != nil {
					w.onComplete(0)
				}
				return nil, 0, WaitInterrupt
			}
	} else {
		a := time.After(timeout)
		for {
			select {
			case data, ok := <-w.data:
				if !ok {
					log.Errorf("Wait chan is closed, msgId=[%v]", w.msgId)
					return nil, 0, nil //ChanIsClosed
				}
				msgId, raw := w.decode(data)
				if w.onComplete != nil {
					w.onComplete(msgId)
				}
				return raw, msgId, nil
			case <-a:
				log.Errorf("Wait wait timeout, msgId=[%v]", w.msgId)
				if w.onComplete != nil {
					w.onComplete(0)
				}
				return nil, 0, WaitTimeout
			case <- w.exitWait:
				log.Infof("get Interrupt sig2")
				if w.onComplete != nil {
					w.onComplete(0)
				}
				return nil, 0, WaitInterrupt
			}
		}
	}
	log.Errorf("Wait unknow error, msgId=[%v]", w.msgId)
	if w.onComplete != nil {
		w.onComplete(0)
	}
	return nil, 0, UnknownError
}

func (w *waiter) post(msgId int64, content []byte) {
	if w == nil {
		return
	}
	w.data <- w.encode(msgId, content)
}

func (w *waiter) reset() {
	w.onComplete = nil
	w.msgId = 0
}
