package tcp

import (
	"time"
	"encoding/binary"
	log "github.com/sirupsen/logrus"
	"fmt"
)
type waiter struct {
	msgId     int64
	data      chan []byte
	time      int64
	onComplete func(int64)
	client *Client
}

func (w *waiter) encode(msgId int64, raw []byte) []byte {
	data := make([]byte, 8 + len(raw))
	binary.LittleEndian.PutUint64(data[:8], uint64(msgId))
	copy(data[8:], raw)
	return data
}

func (w *waiter) decode(data []byte) (int64, []byte) {
	msgId := int64(binary.LittleEndian.Uint64(data[:8]))
	return msgId, data[8:]
}

// 如果timeout <= 0 永不超时
func (w *waiter) Wait(timeout time.Duration) ([]byte, int64, error) {
	tick := time.NewTicker(10 * time.Millisecond)
	// if timeout is 0, never timeout
	if timeout <= 0 {
		for {
			select {
			case data, ok := <-w.data:
				if !ok {
					return nil, 0, nil //ChanIsClosed
				}
				msgId, raw := w.decode(data)
				w.onComplete(msgId)
				return raw, msgId, nil
			case <-tick.C:
				if w.client.status & statusConnect <= 0 {
					w.onComplete(0)
					return nil, 0, NetWorkIsClosed
				}
			}
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
				w.onComplete(msgId)
				return raw, msgId, nil
			case <-a:
				log.Errorf("Wait wait timeout, msgId=[%v]", w.msgId)
				w.onComplete(0)
				return nil, 0, WaitTimeout
			case <-tick.C:
				if w.client.status & statusConnect <= 0 {
					w.onComplete(0)
					return nil, 0, NetWorkIsClosed
				}
			}
		}
	}
	log.Errorf("Wait unknow error, msgId=[%v]", w.msgId)
	w.onComplete(0)
	return nil, 0, UnknownError
}

func newWaiter(client *Client, msgId int64, onComplete func(i int64)) *waiter {
	if onComplete == nil {
		onComplete = func(i int64) {
			// just for some debug
			fmt.Println("######", i, " is complete######")
		}
	}
	return &waiter{
		msgId: msgId,
		data:  make(chan []byte, 1),
		time:  int64(time.Now().UnixNano() / 1000000),
		onComplete: onComplete,
		client: client,
	}
}

