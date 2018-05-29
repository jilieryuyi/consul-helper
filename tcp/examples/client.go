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
			d := "GET / HTTP/1.1\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8\r\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36\r\nHost: www.baidu.com\r\nAccept-Language:zh-cn\r\n\r\nhello"
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
