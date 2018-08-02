package tcp

import (
	"testing"
	"bytes"
	"fmt"
)

// go test -v -test.run TestCodec_Encode
func TestCodec_Encode(t *testing.T) {
	msgId := int64(1)
	data  := []byte("hello")
	cc    := Encode(msgId, data)
	fmt.Println(cc)
	mid, c, _ , err := Decode(cc)
	fmt.Println(mid, c, err)
	if err != nil {
		t.Errorf(err.Error())
	}
	if mid != msgId {
		t.Error("error")
	}
	if !bytes.Equal(c, data) {
		t.Error("error 2")
	}

	// 异常包解析
	mid, c, _, err = Decode([]byte("你好"))
	fmt.Println(mid, c, err)
	// 返回的err不应该是nil
	if err == nil {
		t.Errorf("decode fail")
	}
	// 消息id不应该相等
	if mid == msgId {
		t.Error("decode")
	}
	// 解析内容
	if bytes.Equal(c, []byte("你好")) {
		t.Error("error 2")
	}

	// 异常包解析
	mid, c, _, err = Decode(nil)
	fmt.Println(mid, c, err)
	// 返回的err不应该是nil
	if err == nil {
		t.Errorf("decode fail")
	}
	// 消息id不应该相等
	if mid == msgId {
		t.Error("decode")
	}
	// 解析内容
	if !bytes.Equal(c, nil) {
		t.Error("error 2")
	}

	// 异常包解析
	mid, c, _, err = Decode(nil)
	fmt.Println(mid, c, err)
	// 返回的err不应该是nil
	if err == nil {
		t.Errorf("decode fail")
	}
	// 消息id不应该相等
	if mid == msgId {
		t.Error("decode")
	}
	// 解析内容
	if !bytes.Equal(c, nil) {
		t.Error("error 2")
	}
}
