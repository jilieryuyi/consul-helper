package tcp

import (
	"testing"
	"bytes"
)

// go test -v -test.run TestNewPackage
func TestNewPackage(t *testing.T) {
	msgId := int64(1)
	data  := []byte("hello")
	cc    := Encode(msgId, data)
	rd    := bytes.NewReader(cc)
	frame := newPackage(rd)

	content, pmsgId, err := frame.parse()
	if err != nil {
		t.Errorf("parse error: %+v", err)
		return
	}
	if !bytes.Equal(data, content) {
		t.Errorf("parse error")
		return
	}
	if pmsgId != msgId {
		t.Errorf("parse error")
		return
	}
}
