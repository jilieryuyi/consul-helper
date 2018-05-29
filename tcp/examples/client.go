package main


import (
	"github.com/jilieryuyi/wing-go/tcp"
	"context"
	"time"
	"fmt"
	"github.com/sirupsen/logrus"
)
func main() {
	// 先运行server端
	// go run server.go
	// 在运行client端
	// go run client
	address := "127.0.0.1:7770"
	client  := tcp.NewClient(context.Background())
	err     := client.Connect(address, time.Second * 3)

	if err != nil {
		logrus.Errorf("connect to %v error: %v", address, err)
	}
	defer client.Disconnect()

	start := time.Now()
	times := 1000
	for i := 0; i < times; i++ {
		w1, _ := client.Send([]byte("hello"))
		w2, _ := client.Send([]byte("word"))
		w3, _ := client.Send([]byte("hahahahahahahahahahah"))
		res1, _ := w1.Wait(time.Second * 3)
		res2, _ := w2.Wait(time.Second * 3)
		res3, _ := w3.Wait(time.Second * 3)

		fmt.Println("w1 return: ", string(res1))
		fmt.Println("w2 return: ", string(res2))
		fmt.Println("w3 return: ", string(res3))
	}
	fmt.Println("avg use time ", time.Since(start).Nanoseconds()/int64(times)/1000000, "ms")
}
