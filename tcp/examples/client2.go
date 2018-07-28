package main

import (
	"time"
	"github.com/sirupsen/logrus"
	"bytes"
	"fmt"
	"github.com/jilieryuyi/wing-go/tcp"
	"os"
	"context"
	"math/rand"

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
func main() {

	address := "127.0.0.1:7771"


	//go func() {
	//	dial := net.Dialer{Timeout: time.Second * 3}
	//	conn, _ := dial.Dial("tcp", address)
	//	for {
	//		// 这里发送一堆干扰数据包
	//		conn.Write([]byte("你好曲儿个人感情如"))
	//	}
	//}()
	times := 100000
	var res1 []byte
	var data1 []byte
	var client *tcp.Client
	var err error
	for  i := 0; i < times; i++ {
		fmt.Println("###################=>11")

		client, err = tcp.NewClient(context.Background(), address, tcp.SetClientConnectTimeout(time.Second * 3))
		fmt.Println("###################=>1")
		errHappend := false
		errStr := ""
		for {
			if err != nil {
				errStr = err.Error()
				errHappend = true
				break
			}
			data1 = []byte(RandString())
			if len(data1) <= 0 {
				break
			}
			fmt.Println("###################=>21")

			w1, _, err := client.Send(data1)
			if err != nil || w1 == nil {
				errStr = err.Error()
				logrus.Errorf("server_test.go TestNewClient  %v", err)
				errHappend = true
				break
			}
			fmt.Println("###################=>2")

			fmt.Println("###################=>31")

			if w1 != nil {
				res1, _, err = w1.Wait(time.Second * 3)
				if err != nil {
					errStr = err.Error()
					logrus.Errorf("server_test.go TestNewClient %v", err)
					errHappend = true
					break
				}
			}
			fmt.Println("###################=>3")

			logrus.Infof("server_test.go TestNewClient send data=[%v, %v]", string(data1), data1)
			logrus.Infof("server_test.go TestNewClient return data=[%v, %v]", string(res1), res1)
			if !bytes.Equal(data1, res1) {
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

