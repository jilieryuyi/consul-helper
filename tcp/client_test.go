package tcp

import (
	"testing"
	"context"
	"time"
	"bytes"
	"math/rand"
	"github.com/sirupsen/logrus"
	"net"
)

func NewTestServer(address string) *Server {
	//address := "127.0.0.1:7771"
	return NewServer(context.Background(), address, SetOnServerMessage(func(node *ClientNode, msgId int64, data []byte) {
		logrus.Infof("server_test.go TestNewServer receive: msgId=%v, data=%v", msgId, string(data))
		_, err := node.Send(msgId, data)
		if err != nil {
			logrus.Errorf("server_test.go TestNewServer line 32: %v", err)
		}
	}))
}

// go test -v -test.run TestNewClient
func TestNewClient(t *testing.T) {
	address := "127.0.0.1:7771"
	//server := NewTestServer(address)
	//server.Start()
	//defer server.Close()
	//time.Sleep(time.Millisecond * 100)
	go func() {
		dial := net.Dialer{Timeout: time.Second * 3}
		conn, err := dial.Dial("tcp", address)
		if err == nil {
			for {
				// 这里发送一堆干扰数据包
				conn.Write([]byte("你好曲儿个人感情如"))
			}
		}
	}()
	var (
		res1  []byte
	 	data1 []byte
	 	client *Client
		times = 10
		err error
		errHappened = false
		errStr = ""
	)
	for  i := 0; i < times; i++ {
		client, err = NewClient(context.Background(), address,
			SetClientConnectTimeout(time.Second * 3),
			SetWaiterTimeout(1000 * 60),
		)
		if err != nil {
			t.Errorf("NewClient error")
			return
		}
		for {
			data1 = []byte(RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1024)))
			if len(data1) <= 0 {
				break
			}
			w1, _, err := client.Send(data1, 0)
			if err != nil || w1 == nil {
				errStr = err.Error()
				t.Errorf("server_test.go TestNewClient  %v", err)
				errHappened = true
				break
			}
			res1, _, err = w1.Wait(0)
			if err != nil {
				errStr = err.Error()
				t.Errorf("server_test.go TestNewClient %v", err)
				errHappened = true
				break
			}
			logrus.Infof("server_test.go TestNewClient send data=[%v, %v]", string(data1), data1)
			logrus.Infof("server_test.go TestNewClient return data=[%v, %v]", string(res1), res1)
			if !bytes.Equal(data1, res1) {
				errStr = "server_test.go TestNewClient error, send != return"
				t.Errorf("server_test.go TestNewClient error, send != return")
				errHappened = true
				break
			}
			break
		}
		client.Close()
		if errHappened {
			t.Errorf(errStr)
			return
		}
	}
}

// go test -v -test.run TestNewClient2
func TestNewClient2(t *testing.T) {
	address := "127.0.0.1:7771"
	//server := NewTestServer(address)
	//server.Start()
	//defer server.Close()
	//go func() {
	//	dial := net.Dialer{Timeout: time.Second * 3}
	//	conn, _ := dial.Dial("tcp", address)
	//	for {
	//		// 这里发送一堆干扰数据包
	//		conn.Write([]byte("你好曲儿个人感情如"))
	//	}
	//}()
	var (
		times = 100
	 	res1 []byte
	 	data1 []byte
	)
	client, err := NewClient(context.Background(), address,
		SetClientConnectTimeout(time.Second * 3),
		SetWaiterTimeout(1000 * 60),
	)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer client.Close()
	for  i := 0; i < times; i++ {
		data1 = []byte(RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1024)))
		if len(data1) <= 0 {
			continue
		}
		w1, _, err := client.Send(data1, 0)
		if err != nil || w1 == nil {
			t.Errorf("server_test.go TestNewClient  %v", err)
			return
		}
		res1, _, err = w1.Wait(time.Second * 3)
		if err != nil {
			t.Errorf("server_test.go TestNewClient %v", err)
			return
		}
		logrus.Infof("server_test.go TestNewClient send data=[%v]", string(data1))
		logrus.Infof("server_test.go TestNewClient return data=[%v]", string(res1))
		if !bytes.Equal(data1, res1) {
			t.Errorf("server_test.go TestNewClient error, send != return")
			return
		}
	}
}