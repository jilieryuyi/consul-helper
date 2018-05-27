package consul

import (
	"testing"
	"fmt"
	"sync"
)

func TestNewService(t *testing.T) {
	address := "127.0.0.1:8500"
	lock    := "test"
	wg := new(sync.WaitGroup)
	wg.Add(1)
	NewService(
		address,
		lock,
		"test1",
		"127.0.0.1",
		7000,
		SetOnLeader(func(isLeader bool) {
			fmt.Println("test1 is leader:", isLeader)
			wg.Done()
		}))
	wg.Add(1)
	NewService(
		address,		lock,

		"test2",
		"127.0.0.1",
		7001,
		SetOnLeader(func(isLeader bool) {
			fmt.Println("test2 is leader:", isLeader)
			wg.Done()
		}))
	wg.Add(1)
	 NewService(
		address,		lock,

		 "test3",
		"127.0.0.1",
		7002,
		SetOnLeader(func(isLeader bool) {
			fmt.Println("test3 is leader:", isLeader)
			wg.Done()
		}))

	wg.Wait()
}
