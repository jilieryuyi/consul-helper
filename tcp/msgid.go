package tcp

import "sync/atomic"

var (
	globalMsgId int64 = 1
)

func getMsgId() int64 {
	msgId := atomic.AddInt64(&globalMsgId, 1)
	// check max msgId
	if msgId > MaxInt64 {
		atomic.StoreInt64(&globalMsgId, 1)
		msgId = atomic.AddInt64(&globalMsgId, 1)
	}
	return msgId
}
