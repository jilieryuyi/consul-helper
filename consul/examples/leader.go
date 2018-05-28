package main

import (
	"github.com/sirupsen/logrus"
	"github.com/jilieryuyi/wing-go/consul"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	serviceName := "service-test"
	lockKey     := "test"
	address     := "127.0.0.1:8500"


	leader1 := consul.NewLeader(address, lockKey, serviceName, "127.0.0.1", 7770)
	leader1.Select(func(member *consul.ServiceMember) {
		logrus.Infof("1=>%+v", member)
	})
	//defer leader1.Free()
	// wait a second, and start anther service
	//time.Sleep(time.Second)

	leader2 := consul.NewLeader(address, lockKey, serviceName, "127.0.0.1", 7771, )
	leader2.Select(func(member *consul.ServiceMember) {
		logrus.Infof("2=>%+v", member)
	})
	//defer leader2.Free()
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