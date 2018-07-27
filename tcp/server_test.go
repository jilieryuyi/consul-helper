package tcp

import (
	"testing"
	"context"
	"time"
	"bytes"
	"fmt"
	//"net"
	"math/rand"
	"github.com/sirupsen/logrus"
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

func TestNewServer(t *testing.T) {
	address := "127.0.0.1:7771"
	server  := NewServer(context.Background(), address, SetOnServerMessage(func(node *ClientNode, msgId int64, data []byte) {
		logrus.Infof("receive: msgId=%v, data=%v", msgId, string(data))
		_, err := node.Send(msgId, data)
		if err != nil {
			logrus.Errorf("line 32: %v", err)
		}
	}))
	server.Start()
	defer server.Close()
	time.Sleep(time.Second)

	//go func() {
	//	dial := net.Dialer{Timeout: time.Second * 3}
	//	conn, _ := dial.Dial("tcp", address)
	//	for {
	//		// 这里发送一堆干扰数据包
	//		conn.Write([]byte("你好曲儿个人感情如"))
	//	}
	//}()


	client  := NewClient(context.Background())

	start := time.Now()
	times := 10
	for  i := 0; i < times; i++ {
		err     := client.Connect(address, time.Second * 3)
		if err != nil {
			t.Errorf("connect to %v error: %v", address, err)
			return
		}
		var res1 []byte
		var res2 []byte
		var res3 []byte

		data1 := []byte(RandString())
		data2 := []byte(RandString())
		data3 := []byte(RandString())
		w1, _, err := client.Send(data1)
		if err != nil || w1 == nil {
			t.Errorf("%v", err)
			return
		}
		w2, _, err := client.Send(data2)
		if err != nil || w2 == nil {
			t.Errorf("%v", err)
			return
		}
		w3, _, err := client.Send(data3)
		if err != nil || w3 == nil {
			t.Errorf("%v", err)
			return
		}
		if w1 != nil {
			res1, err = w1.Wait(time.Second * 3)
			if err != nil {
				t.Errorf("%v", err)
				return
			}
		}
		if w2 != nil {
			res2, err = w2.Wait(time.Second * 3)
			if err != nil {
				t.Errorf("%v", err)
				return
			}
		}
		if w3 != nil {
			res3, err = w3.Wait(time.Second * 3)
			if err != nil {
				t.Errorf("%v", err)
				return
			}
		}

		logrus.Infof("data1 == %v", string(data1))
		logrus.Infof("data2 == %v", string(data2))
		logrus.Infof("data3 == %v", string(data3))

		logrus.Infof("res1 == %v", string(res1))
		logrus.Infof("res2 == %v", string(res2))
		logrus.Infof("res3 == %v", string(res3))

		if !bytes.Equal(data1, res1) || !bytes.Equal(data2, res2) || !bytes.Equal(data3, res3) {
			t.Errorf("error")
			return
		}
		fmt.Println("w1 return: ", string(res1))
		fmt.Println("w2 return: ", string(res2))
		fmt.Println("w3 return: ", string(res3))
		client.Disconnect()
	}
	fmt.Println("avg use time ", time.Since(start).Nanoseconds()/int64(times), "ns")
}