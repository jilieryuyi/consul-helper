package main

import (
	"github.com/jilieryuyi/wing-go/tcp"
)

type coder struct {

}

func (c coder) Encode(msg []byte) []byte {
	return msg
}

func (c coder) Decode(data []byte) ([]byte, int, error) {
	return data, len(data), nil
}

func main() {
	client := NewClient()
}
