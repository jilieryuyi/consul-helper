package tcp

import (
	"errors"
	"fmt"
)
var (
	NotConnect    = errors.New("not connect")
	IsConnected   = errors.New("is connected")
	WaitTimeout   = errors.New("wait timeout")
	UnknownError  = errors.New("unknown error")
	WaitInterrupt = errors.New("wait interrupt")
)

type Error struct {
	err error
	errCode int32
}

func NewError(errorCode int32, err error) error {
	return &Error{err: err, errCode: errorCode}
}
func (e *Error) ErrorCode() int32 {
	return e.errCode
}
func (e *Error) Error() string{
	if e.err != nil {
		return fmt.Sprintf("%d %s %v",
			e.errCode, ErrName(e.errCode), e.err)
	} else {
		return fmt.Sprintf("%d %s",
			e.errCode, ErrName(e.errCode))
	}
}

func ErrName(err int32) string {
	switch err {
	default:
		return fmt.Sprintf("not implement %d", err)
	}
	return fmt.Sprintf("not implement %d", err)
}
