package tcp

import (
	"testing"
	"time"
	"bytes"
	"fmt"
)

// go test -v -test.run TestWaiter_Wait
func TestWaiter_Wait(t *testing.T) {

	wai := newWaiter(100, func(i int64) {
		fmt.Println("######", i, " is complete######")
	})

	// 没有数据写入 data chan， 1秒后超时，应该返回错误
	// 次数byt应该为nil，err不为nil
	byt, msgId, err := wai.Wait(time.Second)
	if err == nil || byt != nil {
		t.Errorf("wait error")
		return
	}

	// 写入数据
	raw := []byte("hello")
	wai.data <-  wai.encode(wai.msgId, raw)

	// 这个时候读到的数据应该是hello，err应该为nil
	byt, msgId, err = wai.Wait(time.Second)
	fmt.Println(string(byt), msgId, err)
	if err != nil {
		t.Errorf("wait error")
		return
	}
	// 如果与原始数据不相等，说明发生了错误
	if !bytes.Equal(byt, raw) {
		t.Errorf("wait error")
		return
	}

	go func() {
		time.Sleep(time.Second * 5)
		wai.data <-  wai.encode(wai.msgId, raw)
	}()

	// 测试永不超时的情况
	byt, msgId, err = wai.Wait(0)
	if err != nil {
		t.Errorf("wait error")
		return
	}
	// 如果与原始数据不相等，说明发生了错误
	if !bytes.Equal(byt, raw) {
		t.Errorf("wait error")
		return
	}

	go func() {
		time.Sleep(time.Second * 5)
		//wai.data <-  wai.encode(wai.msgId, raw)
		wai.StopWait()
	}()

	// 测试永不超时的情况
	// 手动中断等待
	// 这里的 err 应该等于 WaitInterrupt
	byt, msgId, err = wai.Wait(0)
	if err == nil {
		t.Errorf("wait error")
		return
	}

}
