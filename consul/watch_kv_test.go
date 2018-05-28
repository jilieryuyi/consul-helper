package consul

import (
	"testing"
	"github.com/hashicorp/consul/api"
	"fmt"
	"sync"
	"bytes"
	"time"
)
func TestNewWatchKv(t *testing.T) {
	config := api.DefaultConfig()
	config.Address = "127.0.0.1:8500"
	value := []byte("hello word")
	wg := new(sync.WaitGroup)
	wg.Add(1)
	client, _ := api.NewClient(config)
	watch := NewWatchKv(client.KV(), "test")
	watch.Watch(func(bt []byte) {
		wg.Done()
		fmt.Println("new data: ", string(bt))
		if !bytes.Equal(value, bt) {
			t.Errorf("watch error")
		}
	})

	kv := NewKvEntity(client.KV(), "test/a", value)
	kv.Set()
	wg.Wait()

	a := time.After(time.Second)
	select {
	case <- a:
		kv.Delete()
	}
}
