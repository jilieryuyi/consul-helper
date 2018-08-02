package main

import (
	"net"
	"time"
	"math/rand"
	"bytes"
	"github.com/jilieryuyi/wing-go/tcp"
	"context"
	"github.com/sirupsen/logrus"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

func TestClient1() {
	address := "127.0.0.1:7771"
	go func() {
		dial := net.Dialer{Timeout: time.Second * 3}
		conn, err := dial.Dial("tcp", address)
		if err == nil {
			for {
				// 这里发送一堆干扰数据包
				// 这里报文没有按照规范进行封包
				// 目的是为了测试服务端的解包容错性
				conn.Write([]byte(tcp.RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(128))))
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
	var (
		res  []byte
		data []byte
		client *tcp.Client
		times = 100000000
		err error
	)
	for  i := 0; i < times; i++ {
		client, err = tcp.NewClient(
			context.Background(),
			address,
			tcp.SetClientConnectTimeout(time.Second * 3),
			tcp.SetWaiterTimeout(1000 * 60),
		)
		if err != nil {
			logrus.Errorf("NewClient error")
			return
		}
		err = nil
		for {
			data = []byte(tcp.RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(128)))
			if len(data) <= 0 {
				break
			}
			wai, _, err := client.Send(data, 0)
			if err != nil {
				logrus.Errorf("Send fail, err=[%v]", err)
				break
			}
			res, _, err = wai.Wait(0)
			if err != nil {
				logrus.Errorf("Wait fail, err=[%v]", err)
				break
			}
			if !bytes.Equal(data, res) {
				logrus.Errorf("send != return")
				err = errors.New("send != return")
				break
			}
			break
		}
		client.Close()
		if err != nil {
			logrus.Errorf(err.Error())
			return
		}
	}
}

// 注意：运行以下测试之前先启动服务端 go run examples/server.go
// go test -v -test.run TestNewClient2
func TestClient2() {
	address := "127.0.0.1:7771"
	go func() {
		dial := net.Dialer{Timeout: time.Second * 3}
		conn, err := dial.Dial("tcp", address)
		if err == nil {
			for {
				// 这里发送一堆干扰数据包
				// 这里报文没有按照规范进行封包
				// 目的是为了测试服务端的解包容错性
				conn.Write([]byte(tcp.RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(128))))
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
	client, err := tcp.NewClient(
		context.Background(),
		address,
		tcp.SetClientConnectTimeout(time.Second * 3),
		tcp.SetWaiterTimeout(1000 * 60),
	)
	if err != nil {
		logrus.Errorf("%v", err)
		return
	}
	defer client.Close()

	var (
		times = 100000000
		res []byte
		data []byte
	)

	for  i := 0; i < times; i++ {
		data = []byte(tcp.RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1024)))
		if len(data) <= 0 {
			continue
		}
		wai, _, err := client.Send(data, 0)
		if err != nil {
			logrus.Errorf("Send fail, err=[%v]", err)
			return
		}
		res, _, err = wai.Wait(time.Second * 3)
		if err != nil {
			logrus.Errorf("Wait fail, err=[%v]", err)
			return
		}
		if !bytes.Equal(data, res) {
			logrus.Errorf("Equal fail, send != return")
			return
		}
	}
}


func main() {
	go TestClient1()
	go TestClient2()

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
