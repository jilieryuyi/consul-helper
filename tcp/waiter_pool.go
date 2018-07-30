package tcp

import "sync"

type waiterPool struct {
	maxSize int64
	pool []*waiter
	lock *sync.Mutex
}

func newWaiterPool(maxSize int64) *waiterPool {
	p := &waiterPool{
		maxSize: maxSize,
		pool: make([]*waiter, 64),
		lock: new(sync.Mutex),
	}
	for i := 0; i < 64; i++ {
		p.pool[i] = newWaiter(0, nil)
	}
	return p
}

func (p *waiterPool) get(msgId int64, oncomplete func(i int64)) (*waiter, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for _, w := range p.pool {
		if w.msgId <= 0 {
			w.msgId = msgId
			w.onComplete = oncomplete
			return w, nil
		}
	}
	if int64(len(p.pool)) < p.maxSize {
		w := newWaiter(msgId, oncomplete)
		p.pool = append(p.pool, w)
		return w, nil
	}
	return nil, ErrPoolMaxSize
}
