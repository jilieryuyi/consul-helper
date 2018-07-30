package tcp

import "testing"

// go test -v -test.run Test_GetMsgId
func Test_GetMsgId(t *testing.T) {
	if 2 != getMsgId() {
		t.Errorf("getMsgId fail")
	}
	if 3 != getMsgId() {
		t.Errorf("getMsgId fail")
	}
}
