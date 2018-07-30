package tcp

import (
	"testing"
	"context"
	"time"
)

// go test -v -test.run Test_newWaiterManager
func Test_newWaiterManager(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newWaiterManager(ctx)
	m.append(newWaiter(1, nil))
	if len(m.waiter) != 1 {
		t.Errorf("WaiterManager append fail")
	}
	// test timeout
	time.Sleep(time.Second * 7)
	if len(m.waiter) > 0 {
		t.Errorf("WaiterManager checktimeout fail")
	}

	m.append(newWaiter(1, nil))
	if len(m.waiter) != 1 {
		t.Errorf("WaiterManager append fail")
	}

	m.clear(1)
	if len(m.waiter) != 0 {
		t.Errorf("WaiterManager clear fail")
	}

	m.append(newWaiter(1, nil))
	if len(m.waiter) != 1 {
		t.Errorf("WaiterManager append fail")
	}

	if nil == m.get(1) {
		t.Errorf("WaiterManager get fail")
	}

	m.clearAll()
	if len(m.waiter) != 0 {
		t.Errorf("WaiterManager clearAll fail")
	}
}
