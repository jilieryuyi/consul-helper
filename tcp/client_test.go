package tcp

import (
	"testing"
	"context"
	"time"
	"bytes"
	"math/rand"
	"net"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
)
// 注意：运行以下测试之前先启动服务端 go run examples/server.go
// 测试连续连接和关闭连接10000次，观看服务器和客户端是否正常
// go test -v -test.run TestNewClient
func TestNewClient(t *testing.T) {
	address := "127.0.0.1:7771"
	go func() {
		dial := net.Dialer{Timeout: time.Second * 3}
		conn, err := dial.Dial("tcp", address)
		if err == nil {
			for {
				// 这里发送一堆干扰数据包
				// 这里报文没有按照规范进行封包
				// 目的是为了测试服务端的解包容错性
				conn.Write([]byte(RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(128))))
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
	var (
		res  []byte
	 	data []byte
	 	client *Client
		times = 10000
		err error
	)
	for  i := 0; i < times; i++ {
		client, err = NewClient(
			context.Background(),
			address,
			SetClientConnectTimeout(time.Second * 3),
			SetWaiterTimeout(1000 * 60),
		)
		if err != nil {
			t.Errorf("NewClient error")
			return
		}
		err = nil
		for {
			data = []byte(RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(128)))
			if len(data) <= 0 {
				break
			}
			wai, _, err := client.Send(data, 0)
			if err != nil {
				t.Errorf("Send fail, err=[%v]", err)
				break
			}
			res, _, err = wai.Wait(0)
			if err != nil {
				t.Errorf("Wait fail, err=[%v]", err)
				break
			}
			if !bytes.Equal(data, res) {
				t.Errorf("send != return")
				err = errors.New("send != return")
				break
			}
			break
		}
		client.Close()
		if err != nil {
			t.Errorf(err.Error())
			return
		}
	}
}

// 注意：运行以下测试之前先启动服务端 go run examples/server.go
// go test -v -test.run TestNewClient2
func TestNewClient2(t *testing.T) {
	address := "127.0.0.1:7771"
	go func() {
		dial := net.Dialer{Timeout: time.Second * 3}
		conn, err := dial.Dial("tcp", address)
		if err == nil {
			for {
				// 这里发送一堆干扰数据包
				// 这里报文没有按照规范进行封包
				// 目的是为了测试服务端的解包容错性
				conn.Write([]byte(RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(128))))
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
	client, err := NewClient(
		context.Background(),
		address,
		SetClientConnectTimeout(time.Second * 3),
		SetWaiterTimeout(1000 * 60),
	)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer client.Close()

	var (
		times = 10000
		res []byte
		data []byte
	)

	for  i := 0; i < times; i++ {
		data = []byte(RandString(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1024)))
		if len(data) <= 0 {
			continue
		}
		wai, _, err := client.Send(data, 0)
		if err != nil {
			t.Errorf("Send fail, err=[%v]", err)
			return
		}
		res, _, err = wai.Wait(time.Second * 3)
		if err != nil {
			t.Errorf("Wait fail, err=[%v]", err)
			return
		}
		if !bytes.Equal(data, res) {
			t.Errorf("Equal fail, send != return")
			return
		}
	}
}

// 对于http客户端，所有的数据编码解码均原封不动直接返回
var invalidPackage = errors.New("invalid package")
var timeout = errors.New("read timeout")
type testCodec struct {}
func (t *testCodec) Encode(msgId int64, msg []byte) []byte {
	return msg
}
// 第一个返回值是消息id，1是预留消息id，这里直接返回2以上的数字即可
func (t *testCodec) Decode(rd io.Reader) ([]byte, int64, error) {
	mark := []byte("\r\n\r\n")
	var buffer = make([]byte, 0)
	var buf = make([]byte, len(mark))

	for {
		select {
		case <- time.After(time.Second * 3):
			return nil, 2, timeout
		default:
			_, err := rd.Read(buf)
			if err != nil {
				fmt.Println("Decode fail, ", err)
				return nil, 2, err
			}
			buffer = append(buffer, buf...)
			//fmt.Println(string(buffer) + "\r\n")
			if bytes.Equal(buf, mark) {
				fmt.Println("find mark")
				// 这时的buffer为http的header
				// 找到包长度 Content-Length: 14543

				reg := regexp.MustCompile(`Content-Length:[\s]{0,}[\d]{1,}(?i)`)
				matchs := reg.FindAllString(string(buffer), -1)
				//fmt.Println(matchs[0])
				if len(matchs) <= 0 {
					return nil, 2, invalidPackage
				}
				reg = regexp.MustCompile(`[\d]{1,}`)
				matchs = reg.FindAllString(matchs[0], -1)
				//fmt.Println(matchs[0])
				if len(matchs) <= 0 {
					return nil, 2, invalidPackage
				}
				fmt.Println("content len=", matchs[0])
				contentLen, err := strconv.ParseInt(matchs[0], 10, 64)
				if err != nil {
					return nil, 2, err
				}
				if contentLen <= 0 {
					return buffer, 2, nil
				}
				var b= make([]byte, int(contentLen))
				_, err = rd.Read(b)
				if err != nil {
					return nil, 2, err
				}
				return buffer, 2, nil
			}
		}
	}
	return buffer, 2, nil
}

// go test -v -test.run Test_GetContentLen
func Test_GetContentLen(t *testing.T) {
	content := ("Content-Encoding: gzip\r\nContent-Length: 14543\r\nContent-Type: application/javascript")
	reg := regexp.MustCompile(`Content-Length:[\s]{0,}[\d]{1,}(?i)`)
	matchs := reg.FindAllString(content, -1)
	fmt.Println(matchs[0])
	reg = regexp.MustCompile(`[\d]{1,}`)
	matchs = reg.FindAllString(matchs[0], -1)
	fmt.Println(matchs[0])
}


// 测试连接百度80端口，ip来自于ping
// test http client
// go test -v -test.run TestNewClient3
func TestNewClient3(t *testing.T) {
	client, err := NewClient(
		context.Background(),
		//http://blog.sina.com.cn/s/blog_1574497330102wkfs.html
		"14.215.177.39:80",
		SetCodec(&testCodec{}),
	)
	if err != nil {
		t.Errorf("NewClient fail, error=[%v]", err)
		return
	}
	defer client.Close()
	data := "GET / HTTP/1.1\r\nHost: www.baidu.com\r\nUser-Agent: Mozilla/5.0 (Windows NT 6.1; WOW64; rv:47.0) Gecko/20100101 Firefox/47.0\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\n\r\n"// +
	//data = "GET / HTTP/1.1\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8\r\n"+
	//	"Accept-Encoding: gzip, deflate, br\r\n"+
	//	"Accept-Language: zh-CN,zh;q=0.9,en;q=0.8\r\n"+
	//	"Cache-Control: max-age=0\r\n"+
	//	"Connection: keep-alive\r\n"+
	//	"Cookie: aliyungf_tc=AQAAACU0wHQ7wQkA/od7d5dZf145OFi5; acw_tc=AQAAANWEXBe+0wkA/od7d2D01nwTtIRL; _ga=GA1.2.2110621287.1533248067; _gid=GA1.2.650556162.1533248067; Hm_lvt_c4fb630bdc21e7a693a06c26ba5651c6=1533248067; gr_user_id=7c459af0-1a0e-4329-afab-d2fd4fe982f1; gr_session_id_b73895aacc73ac75=0af5a8b6-8a30-4528-8e42-bbd9e9614733; gr_session_id_b73895aacc73ac75_0af5a8b6-8a30-4528-8e42-bbd9e9614733=true; _gat=1; Hm_lpvt_c4fb630bdc21e7a693a06c26ba5651c6=1533248127\r\n"+
	//	"Host: www.yonglibao.com\r\n"+
	//	"If-Modified-Since: Wed, 11 Jul 2018 10:17:12 GMT\r\n"+
	//	"If-None-Match: W/\"5b45d928-567f\"\r\n"+
	//	"Upgrade-Insecure-Requests: 1\r\n"+
	//	"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.99 Safari/537.36\r\n\r\n"
	wai, _, err := client.Send([]byte(data), time.Second * 6)
	if err != nil {
		t.Errorf("Send fail, error=[%v]", err)
		return
	}
	body, _, err := wai.Wait(time.Second * 6)
	fmt.Println(string(body))
	if err != nil {
		t.Errorf("Wait fail, error=[%v]", err)
		return
	}
}