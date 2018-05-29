package main

import (
	"github.com/jilieryuyi/wing-go/tcp"
	"fmt"
	"time"
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
	client := tcp.NewSyncClient(
		"14.215.177.39",
		80,
		tcp.SetWriteTimeout(time.Second * 3),
		tcp.SetReadTimeout(time.Second * 3),
		tcp.SetConnectTimeout(time.Second),
		tcp.SetSyncCoder(&coder{}),
	)
	err := client.Connect()
	if err != nil {
		fmt.Println("error: ", err)
		return
	}
	defer client.Disconnect()
	d := "GET /index.html HTTP/1.1\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8\r\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36\r\nHost: www.baidu.com\r\nAccept-Language:zh-cn\r\n\r\nhello"
	res, e := client.Send([]byte(d))
	fmt.Println(string(res), e)
}
