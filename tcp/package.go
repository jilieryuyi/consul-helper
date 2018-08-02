package tcp

import (
	"io"
	"net"
	"bufio"
	"encoding/binary"
	"bytes"
	"github.com/pkg/errors"
)

type packageFrame struct {
	rd io.Reader
	header []byte
	packageLen []byte
	msgId []byte
}
var ErrReadNotComplete = errors.New("read not complete")
var ErrInvalidHeader = errors.New("invalid package header")
const (
	headerLen = 4
	packageLen = 4
	msgIdLen = 8
)
func newPackage(conn *net.Conn) *packageFrame {
	rd := bufio.NewReader(*conn)
	return &packageFrame{
		rd: rd,
		header: make([]byte, headerLen),
		packageLen: make([]byte, packageLen),
		msgId: make([]byte, msgIdLen),
	}
}

func (p *packageFrame) readHeader() error {
	n, err := p.rd.Read(p.header)
	if err != nil {
		return err
	}
	if n < headerLen {
		return ErrReadNotComplete
	}
	if !bytes.Equal(p.header, PackageHeader) {
		return ErrInvalidHeader
	}
	return nil
}

func (p *packageFrame) readPackageLen() (int, error) {
	n, err := p.rd.Read(p.packageLen)
	if err != nil {
		return 0, err
	}
	if n < packageLen {
		return 0, ErrReadNotComplete
	}
	clen := int(binary.LittleEndian.Uint32(p.packageLen))
	return clen, nil
}

func (p *packageFrame) readMsgId() (int64, error) {
	n, err := p.rd.Read(p.msgId)
	if err != nil {
		return 0, err
	}
	if  n < 8 {
		return 0, ErrReadNotComplete
	}
	msgId   := int64(binary.LittleEndian.Uint64(p.msgId))
	return msgId, nil
}

func (p *packageFrame) readContent(clen int) ([]byte, error) {
	//clen, err := p.readPackageLen()
	var content = make([]byte, clen)
	n, err := p.rd.Read(content)
	if err != nil {
		return nil, err
	}
	if n < clen {
		return nil, ErrReadNotComplete
	}
	return content, nil
}


func (p *packageFrame) parse() ([]byte, int64, error)  {
	err := p.readHeader()
	if err != nil {
		return nil, 0, err
	}
	clen, err := p.readPackageLen()
	if err != nil {
		return nil, 0, err
	}
	msgId, err := p.readMsgId()
	if err != nil {
		return nil, 0, err
	}
	content, err := p.readContent(clen)
	if err != nil {
		return nil, 0, err
	}
	return content, msgId, nil
}