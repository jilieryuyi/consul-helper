package tcp

import (
	"testing"
	"bytes"
	"fmt"
	"strings"
	"encoding/binary"
)

// go test -v -test.run TestCodec_Encode
func TestCodec_Encode(t *testing.T) {
	// 正常包解析测试
	msgId := int64(1)
	data  := []byte("hello")
	cc    := Encode(msgId, data)
	mid, c, _ , err := Decode(cc)
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

	// 测试MaxPackError
	data = []byte(strings.Repeat("1", PackageMaxLength+1))
	data[0] = byte(255)
	data[1] = byte(255)
	data[2] = byte(255)
	data[3] = byte(255)
	mid, c, _, err = Decode(data)
	if err != MaxPackError {
		t.Errorf("Decode fail, MaxPackError")
	}

	// 测试InvalidPackage
	data = []byte(strings.Repeat("1", 6))
	data[0] = byte(255)
	data[1] = byte(255)
	data[2] = byte(255)
	data[3] = byte(255)
	mid, c, _, err = Decode(data)
	if err != InvalidPackage {
		t.Errorf("Decode fail, InvalidPackage")
	}

	// 测试InvalidPackage
	data = []byte(strings.Repeat("1", 16))
	data[0] = byte(255)
	data[1] = byte(255)
	data[2] = byte(255)
	data[3] = byte(255)
	binary.LittleEndian.PutUint32(data[4:8], uint32(1))
	mid, c, _, err = Decode(data)
	if err != InvalidPackage {
		t.Errorf("Decode fail, InvalidPackage")
	}
}
