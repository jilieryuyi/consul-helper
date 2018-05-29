package tcp

import (
	"errors"
	"encoding/binary"
	log "github.com/sirupsen/logrus"
)

func Pack(cmd int, msg []byte) []byte {
	l  := len(msg)
	r  := make([]byte, l + 6)
	cl := l + 2
	binary.LittleEndian.PutUint32(r[:4], uint32(cl))
	binary.LittleEndian.PutUint16(r[4:6], uint16(cmd))
	copy(r[6:], msg)
	return r
}

var DataLenError = errors.New("data len error")
// return cmd, content, endPoint, error
func Unpack(data []byte) (int, []byte, int, error) {

	if data == nil || len(data) == 0 {
		return 0, nil, 0, nil
	}
	//defer func() {
	//	if err := recover(); err != nil {
	//		log.Errorf("Unpack recover##########%+v", err)
	//		data = make([]byte, 0)
	//	}
	//}()
	//log.Debugf("data: %+v", *data)
	if len(data) > MAX_PACKAGE_LEN {
		log.Errorf("max len error: %+v", data)
		//*data = make([]byte, 0)
		return 0, nil, 0, MaxPackError
	}
	if len(data) < 6 {
		//log.Warnf("package is not complete")
		return 0, nil, 0, nil
	}
	clen := int(binary.LittleEndian.Uint32(data[:4]))
	if clen < 2 {
		log.Errorf("clen is min the 2")
		return 0, nil, 0, DataLenError
	}
	//log.Debugf("clen=%+v", clen)
	if len(data) < clen + 4 {
		//log.Warnf("package is not complete")
		log.Warnf("data len error %v < %v : %+v\r\n%v", len(data), clen + 4, data, string(data))
		//*data = make([]byte, 0)
		return 0, nil, 0, nil//DataLenError
	}
	//log.Debugf("cmd=%+v", (*data)[4:6])
	cmd     := int(binary.LittleEndian.Uint16(data[4:6]))
	//log.Debugf("content=%+v === %v", (*data)[6 : clen + 4], string((*data)[6 : clen + 4]))
	content := make([]byte, len(data[6 : clen + 4]))
	//content := data[6 : clen + 4]
	copy(content, data[6 : clen + 4])

	//if len(*data) < clen + 4 {
	//	log.Errorf("package error")
	//	*data  = append((*data)[:0], (*data)[len(*data):]...)
	//} else {
	//log.Debugf("data==%+v, %+v==%+v",clen+4, *data, string(*data))

	//*data  = append((*data)[:0], (*data)[clen+4:]...)

	//}
	//tcp.buffer = append(tcp.buffer[:0], tcp.buffer[end:]...)
	//log.Debugf("return(%+v)(%+v)(%+v)", cmd, content, nil)
	return cmd, content, clen + 4, nil
}
