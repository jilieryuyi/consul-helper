package tcp

import (
	"encoding/binary"
	"errors"
)
const MAX_PACKAGE_LEN = 1024000
var MaxPackError = errors.New("package len max then limit")
type ICodec interface {
	Encode(msg []byte) []byte
	Decode(data []byte) ([]byte, int, error)
}
type Codec struct {}
func (c Codec) Encode(msg []byte) []byte {
	l  := len(msg)
	r  := make([]byte, l + 6)
	cl := l + 2
	binary.LittleEndian.PutUint32(r[:4], uint32(cl))
	copy(r[4:], msg)
	return r
}

// 这里的第一个返回值是解包之后的实际报内容
// 第二个返回值是读取了的包长度
func (c Codec) Decode(data []byte) ([]byte, int, error) {
	// 如果解压内容为空
	if data == nil || len(data) == 0 {
		return nil, 0, nil
	}
	// 超过最大允许的包长度
	if len(data) > MAX_PACKAGE_LEN {
		return nil, 0, MaxPackError
	}
	// 这里可以认为是tcp的包不完整
	// 一般不做任何处理
	if len(data) < 4 {
		return nil, 0, nil
	}
	clen := int(binary.LittleEndian.Uint32(data[:4]))
	// 这里可以认为是tcp的包不完整
	// 一般不做任何处理
	if len(data) < clen + 4 {
		return nil, 0, nil
	}
	content := make([]byte, len(data[4 : clen + 4]))
	copy(content, data[4 : clen + 4])
	return content, clen + 4, nil
}

