package tcp

import (
	"testing"
	"time"
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
)

// go test -v -test.run TestWaiter_Wait
func TestWaiter_Wait(t *testing.T) {

	wai := newWaiter(100, func(i int64) {
		fmt.Println("######", i, " is complete######")
	})

	// 没有数据写入 data chan， 1秒后超时，应该返回错误
	// 次数byt应该为nil，err不为nil
	// 测试超时
	byt, msgId, err := wai.Wait(time.Second)
	if err == nil || byt != nil || err != WaitTimeout  {
		t.Errorf("wait error")
		return
	}

	// 测试正常的数据写入流程
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


	// 延迟5秒写入数据，测试永不超时的情况
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


	// 5秒后手动中断，测试手动中断等待
	go func() {
		time.Sleep(time.Second * 5)
		wai.StopWait()
	}()
	// 测试永不超时的情况
	// 手动中断等待
	// 这里的 err 应该等于 WaitInterrupt
	byt, msgId, err = wai.Wait(0)
	if err != WaitInterrupt {
		t.Errorf("wait error")
		return
	}

}

// go test -v -test.run TestWaiter_encode
func TestWaiter_encode(t *testing.T) {
	wai := newWaiter(100, func(i int64) {
		fmt.Println("######", i, " is complete######")
	})
	msgId := int64(100)
	data := []byte("hello")

	encodeData := wai.encode(msgId, data)
	decodeMsgId, decodeData := wai.decode(encodeData)
	logrus.Infof("%v, %v", msgId, data)
	logrus.Infof("%v, %v", decodeMsgId, decodeData)
	if msgId != decodeMsgId || !bytes.Equal(data, decodeData) {
		t.Errorf("encode and decode fail")
	}
}
