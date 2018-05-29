package tcp

import (
	"encoding/binary"
	"errors"
)
const MAX_PACKAGE_LEN = 1024000
var MaxPackError = errors.New("package len max then limit")
type ICodec interface {
	Encode(msgId int64, msg []byte) []byte
	Decode(data []byte) (int64, []byte, int, error)
}
type Codec struct {}
func (c Codec) Encode(msgId int64, msg []byte) []byte {
	l  := len(msg)
	r  := make([]byte, l + 6)
	cl := l + 2
	binary.LittleEndian.PutUint32(r[:4], uint32(cl))
	binary.LittleEndian.PutUint64(r[4:12], uint64(msgId))
	copy(r[12:], msg)
	return r
}

// 这里的第一个返回值是解包之后的实际报内容
// 第二个返回值是读取了的包长度
func (c Codec) Decode(data []byte) (int64, []byte, int, error) {
	if data == nil || len(data) == 0 {
		return 0, nil, 0, nil
	}
	if len(data) > MAX_PACKAGE_LEN {
		return 0, nil, 0, MaxPackError
	}
	if len(data) < 12 {
		return 0, nil, 0, nil
	}
	clen := int(binary.LittleEndian.Uint32(data[:4]))
	if clen < 8 {
		return 0, nil, 0, DataLenError
	}
	if len(data) < clen + 4 {
		return 0, nil, 0, nil
	}
	cmd     := int(binary.LittleEndian.Uint64(data[4:12]))
	content := make([]byte, len(data[12 : clen + 4]))
	copy(content, data[12 : clen + 4])
	return int64(cmd), content, clen + 4, nil
}

