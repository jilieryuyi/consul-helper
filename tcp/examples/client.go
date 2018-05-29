package main

import (
	"github.com/jilieryuyi/wing-go/tcp"
	"context"
	"os"
	"os/signal"
	"syscall"
	"fmt"
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
	client := tcp.NewClient(context.Background(),
		"14.215.177.39",
			80, tcp.SetOnMessage(func(tcp *tcp.Client, content []byte) {
			fmt.Println(string(content))
		}), tcp.SetOnConnect(func(tcp *tcp.Client) {
			d := "GET / HTTP/1.1\r\nAccept: */*"
			n, e:= tcp.Write([]byte(d))
			fmt.Println(n, e)
		}))
	go client.Connect()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sc
}
