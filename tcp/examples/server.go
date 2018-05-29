package main

import (
	"github.com/jilieryuyi/wing-go/tcp"
	"context"
	"os"
	"os/signal"
	"syscall"
)
func main() {
	address := "127.0.0.1:7770"
	server  := tcp.NewAgentServer(context.Background(), address, tcp.SetOnServerMessage(func(node *tcp.TcpClientNode, msgId int64, data []byte) {
		node.Send(msgId, data)
	}))
	server.Start()
	defer server.Close()

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