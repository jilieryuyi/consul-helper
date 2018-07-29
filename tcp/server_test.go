package tcp

import (
	"testing"
	"context"
	"time"
	"bytes"
	//"net"
	"math/rand"
	"github.com/sirupsen/logrus"
	"os"
	"fmt"
)

func RandString() string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bt := []byte(str)
	result := make([]byte, 0)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	slen := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1024)
	for i := 0; i < slen; i++ {
		result = append(result, bt[r.Intn(len(bt))])
	}
	return string(result)
}

// go test -v -test.run TestNewServer
func TestNewServer(t *testing.T) {
	address := "127.0.0.1:7771"
	server  := NewServer(context.Background(), address, SetOnServerMessage(func(node *ClientNode, msgId int64, data []byte) {
		logrus.Infof("server_test.go TestNewServer receive: msgId=%v, data=%v", msgId, string(data))
		_, err := node.Send(msgId, data)
		if err != nil {
			logrus.Errorf("server_test.go TestNewServer line 32: %v", err)
		}
	}))
	defer server.Close()
	server.Start()
}

// go test -v -test.run TestNewClient
func TestNewClient(t *testing.T) {

	address := "127.0.0.1:7771"


	//go func() {
	//	dial := net.Dialer{Timeout: time.Second * 3}
	//	conn, _ := dial.Dial("tcp", address)
	//	for {
	//		// 这里发送一堆干扰数据包
	//		conn.Write([]byte("你好曲儿个人感情如"))
	//	}
	//}()
	times := 10000
	var res1 []byte
	var data1 []byte
	var client *Client
	for  i := 0; i < times; i++ {
		client, _ = NewClient(context.Background(), address, SetClientConnectTimeout(time.Second * 3))
		errHappend := false
		errStr := ""
		for {
			data1 = []byte(RandString())
			if len(data1) <= 0 {
				break
			}
			w1, _, err := client.Send(data1, 0)
			if err != nil || w1 == nil {
				errStr = err.Error()
				t.Errorf("server_test.go TestNewClient  %v", err)
				errHappend = true
				break
			}
			if w1 != nil {
				res1, _, err = w1.Wait(0)
				if err != nil {
					errStr = err.Error()
					t.Errorf("server_test.go TestNewClient %v", err)
					errHappend = true
					break
				}
			}
			logrus.Infof("server_test.go TestNewClient send data=[%v, %v]", string(data1), data1)
			logrus.Infof("server_test.go TestNewClient return data=[%v, %v]", string(res1), res1)
			if !bytes.Equal(data1, res1) {
				errStr = "server_test.go TestNewClient error, send != return"
				t.Errorf("server_test.go TestNewClient error, send != return")
				errHappend = true
				break
			}
			break
		}
		client.Close()
		if errHappend {
			fmt.Println(errStr)
			os.Exit(1)
			return
		}
	}
}

// go test -v -test.run TestNewClient2
func TestNewClient2(t *testing.T) {

	address := "127.0.0.1:7771"


	//go func() {
	//	dial := net.Dialer{Timeout: time.Second * 3}
	//	conn, _ := dial.Dial("tcp", address)
	//	for {
	//		// 这里发送一堆干扰数据包
	//		conn.Write([]byte("你好曲儿个人感情如"))
	//	}
	//}()
	times := 10000
	var res1 []byte
	var data1 []byte
	var client *Client
	client, _ = NewClient(context.Background(), address, SetClientConnectTimeout(time.Second * 3))
	defer client.Close()
	for  i := 0; i < times; i++ {
		errHappend := false
		for {
			data1 = []byte(RandString())
			if len(data1) <= 0 {
				break
			}
			w1, _, err := client.Send(data1, 0)
			if err != nil || w1 == nil {
				t.Errorf("server_test.go TestNewClient  %v", err)
				errHappend = true
				break
			}
			if w1 != nil {
				res1, _, err = w1.Wait(time.Second * 3)
				if err != nil {
					t.Errorf("server_test.go TestNewClient %v", err)
					errHappend = true
					break
				}
			}
			logrus.Infof("server_test.go TestNewClient send data=[%v]", string(data1))
			logrus.Infof("server_test.go TestNewClient return data=[%v]", string(res1))
			if !bytes.Equal(data1, res1) {
				t.Errorf("server_test.go TestNewClient error, send != return")
				errHappend = true
				break
			}
			break
		}
		if errHappend {
			return
		}
	}
}