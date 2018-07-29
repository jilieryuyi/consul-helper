package tcp

import "errors"
var (
	NotConnect    = errors.New("not connect")
	IsConnected   = errors.New("is connected")
	WaitTimeout   = errors.New("wait timeout")
	UnknownError  = errors.New("unknown error")
	WaitInterrupt = errors.New("wait interrupt")
)
