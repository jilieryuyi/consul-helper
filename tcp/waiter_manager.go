package tcp

import (
	"sync"
	"time"
	log "github.com/sirupsen/logrus"
)

type waiterManager struct {
	waiter map[int64]*waiter
	waiterLock *sync.RWMutex
	waiterGlobalTimeout int64 // 毫秒
}

func newWaiterManager() *waiterManager {
	return &waiterManager{
		waiter:              make(map[int64]*waiter),
		waiterLock:          new(sync.RWMutex),
		waiterGlobalTimeout: defaultWaiterTimeout,
	}
}

func (m *waiterManager) append(wai *waiter) {
	m.waiterLock.Lock()
	m.waiter[wai.msgId] = wai
	m.waiterLock.Unlock()
}

func (m *waiterManager) clear(msgId int64) {
	m.waiterLock.Lock()
	wai, ok := m.waiter[msgId]
	if ok {
		delete(m.waiter, msgId)
		wai.StopWait()
		wai.msgId = 0
		wai.onComplete = nil
	}
	m.waiterLock.Unlock()
}

func (m *waiterManager) clearTimeout() {
	current := int64(time.Now().UnixNano() / 1000000)
	m.waiterLock.Lock()
	for msgId, v := range m.waiter  {
		// check timeout
		if current - v.time >= m.waiterGlobalTimeout {
			log.Warnf("Client::keep, msgid=[%v] is timeout, will delete", msgId)
			//close(v.data)
			delete(m.waiter, msgId)
			v.StopWait()
			v.msgId = 0
			v.onComplete = nil
			//tcp.wg.Done()
			// 这里为什么不能使用delWaiter的原因是
			// tcp.waiterLock已加锁，而delWaiter内部也加了锁
			// tcp.delWaiter(msgId)
		}
	}
	m.waiterLock.Unlock()
}

func (m *waiterManager) get(msgId int64) *waiter {
	m.waiterLock.RLock()
	wai, ok := m.waiter[msgId]
	m.waiterLock.RUnlock()
	if ok {
		return wai
	}
	log.Errorf("waiterManager get waiter not found, msgId=[%v]", msgId)
	return nil
}

func (m *waiterManager) clearAll() {
	m.waiterLock.Lock()
	for msgId, v := range m.waiter  {
		log.Infof("clearAll, %v stop wait", msgId)
		v.StopWait()
		v.onComplete = nil
		v.msgId = 0
		//close(v.data)
		delete(m.waiter, msgId)
	}
	m.waiterLock.Unlock()
}
