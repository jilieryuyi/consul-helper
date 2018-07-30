package tcp

import (
	"testing"
	"fmt"
)

// go test -v -test.run Test_WaiterPool_Get
func Test_WaiterPool_Get(t *testing.T) {
	pool := newWaiterPool(1024)
	for i := 0; i < 1026; i++ {
		_, err := pool.get(int64(i), nil)
		fmt.Println(i, err)
		if i < 1024 && err != nil {
			t.Errorf("pool fail: %v", err)
			return
		}
		if i >= 1024 && err == nil {
			t.Errorf("pool fail")
			return
		}
	}
}
