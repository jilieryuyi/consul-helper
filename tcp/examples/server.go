package main

import (
	"github.com/jilieryuyi/wing-go/tcp"
	"context"
	"time"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)
func main() {
	address := "127.0.0.1:7770"
	server := tcp.NewAgentServer(context.Background(), address, tcp.SetOnServerEvents(func(node *tcp.TcpClientNode, event int, data []byte) {
		sendData := tcp.Pack(event, data)
		node.Send(sendData)
	}))
	server.Start()

	time.Sleep(time.Second)
	client := tcp.NewClient(context.Background(), "127.0.0.1", 7770)
	go client.Connect()
	time.Sleep(time.Second)

	w1, _   := client.Send([]byte("hello"))
	w2, _   := client.Send([]byte("word"))
	res1, _ := w1.Wait(time.Second * 3)
	res2, _ := w2.Wait(time.Second * 3)
	client.Disconnect()

	// res1 should be hello
	fmt.Println("w1 return: ", string(res1))
	// res2 should be word
	fmt.Println("w2 return: ", string(res2))

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