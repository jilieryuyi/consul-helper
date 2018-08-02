package tcp

import (
	"encoding/binary"
	"errors"
	"bytes"
	"github.com/sirupsen/logrus"
)

const (
	PackageMaxLength = 1024000
	PackageMinLength = 16
)
var (
	MaxPackError = errors.New("package len max then limit")
	InvalidPackage = errors.New("invalid package")
	PackageHeader = []byte{255, 255, 255, 255}
)

func Encode(msgId int64, msg []byte) []byte {
	// 为了增强容错性，这里加入4字节的header支持

	// 【4字节header长度】 【4字节的内容长度】 【8自己的消息id】 【实际的内容】
	l  := len(msg)
	r  := make([]byte, 4 + l + 4 + 8)

	r[0] = byte(255)
	r[1] = byte(255)
	r[2] = byte(255)
	r[3] = byte(255)

	binary.LittleEndian.PutUint32(r[4:8], uint32(l))
	binary.LittleEndian.PutUint64(r[8:16], uint64(msgId))
	copy(r[16:], msg)
	return r
}

// 这里的第一个返回值是解包之后的实际报内容
// 第二个返回值是读取了的包长度
func Decode(data []byte) (int64, []byte, int, error) {
	if data == nil || len(data) == 0 {
		return 0, nil, 0, InvalidPackage
	}
	startPos := 4
	if !bytes.Equal(data[:4], PackageHeader) {
		i := bytes.Index(data, PackageHeader)
		if i < 0 {
			// 没有找到header，说明这个包为非法包，可以丢弃
			return 0, nil, 0, InvalidPackage
		}
		startPos = i + 4
	}
	if len(data) > PackageMaxLength {
		logrus.Infof("max len error")
		return 0, nil, 0, MaxPackError
	}
	if len(data) < PackageMinLength {
		return 0, nil, 0, nil
	}
	clen := int(binary.LittleEndian.Uint32(data[startPos:startPos+4]))
	if len(data) < clen + 16 {
		return 0, nil, 0, nil
	}
	msgId   := int64(binary.LittleEndian.Uint64(data[startPos+4:startPos+12]))
	content := make([]byte, len(data[startPos+12 : startPos + clen + 12 ]))
	copy(content, data[startPos+12 : startPos + clen + 12])
	return msgId, content, startPos + clen + 12, nil
}
