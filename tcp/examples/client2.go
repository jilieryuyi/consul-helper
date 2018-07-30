package main

import (
	"time"
	"github.com/sirupsen/logrus"
	"bytes"
	"fmt"
	"github.com/jilieryuyi/wing-go/tcp"
	"os"
	"context"
	"strings"
	"math/rand"
	"net"
)

// 这里的客户端测试目的为了测试直连，然后直接关闭这种情形
// 然后观察server端和client端的运行情况
func main() {
	address := "127.0.0.1:7771"
	go func() {
		dial := net.Dialer{Timeout: time.Second * 3}
		conn, _ := dial.Dial("tcp", address)
		for {
			// 这里发送一堆干扰数据包
			conn.Write([]byte(tcp.RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1024))))
		}
	}()
	var (
		times = 1000
		res1 []byte
		data1 []byte
		client *tcp.Client
		err error
	)
	for  i := 0; i < times; i++ {
		fmt.Println("###################=>11")

		client, err = tcp.NewClient(context.Background(), address, tcp.SetClientConnectTimeout(time.Second * 3))
		if err != nil && strings.Index(err.Error(), "too many open files") > 0 {
			logrus.Errorf("%+v", err)
			time.Sleep(time.Second * 3)
			continue
		}
		fmt.Println("###################=>1")
		errHappend := false
		errStr := ""
		for {
			if err != nil {
				errStr = err.Error()
				errHappend = true
				break
			}
			data1 = []byte(tcp.RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(32)))
			if len(data1) <= 0 {
				break
			}
			fmt.Println("###################=>21")

			w1, _, err := client.Send(data1, 0)
			if err != nil || w1 == nil {
				errStr = err.Error()
				logrus.Errorf("server_test.go TestNewClient  %v", err)
				errHappend = true
				break
			}
			fmt.Println("###################=>2")

			fmt.Println("###################=>31")

			if w1 != nil {
				// todo 如果这个时候断网了，消息岂不是永远收不到
				// 如不超时不现实
				// 如果这个时候网络断开了，返回的错误可以忽略
				res1, _, err = w1.Wait(time.Second * 3)
				if err != nil && err != tcp.WaitInterrupt {
					errStr = err.Error()
					logrus.Errorf("server_test.go TestNewClient %v", err)
					errHappend = true
					break
				}
			}
			fmt.Println("###################=>3")

			logrus.Infof("server_test.go TestNewClient send data=[%v, %v]", string(data1), data1)
			logrus.Infof("server_test.go TestNewClient return data=[%v, %v]", string(res1), res1)
			if !bytes.Equal(data1, res1) && err != tcp.WaitInterrupt {
				errStr = "server_test.go TestNewClient error, send != return"
				logrus.Errorf("server_test.go TestNewClient error, send != return")
				errHappend = true
				break
			}
			break
		}
		fmt.Println("###################=>41")
		client.Close()
		fmt.Println("###################=>4")

		if errHappend {
			fmt.Println("error:", errStr)
			os.Exit(1)
			return
		}
	}
	fmt.Println("test ok")
}

