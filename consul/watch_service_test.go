package consul

import (
	"testing"
	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

func TestNewWatchService(t *testing.T) {
	config := api.DefaultConfig()
	config.Address = "127.0.0.1:8500"

	client, _ := api.NewClient(config)
	serviceName := "service-test"
	wg := new(sync.WaitGroup)
	wg.Add(3)
	watch := NewWatchService(client.Health(), serviceName)
	watch.Watch(func(event int, member *ServiceMember) {
		wg.Done()
		switch event {
		case EventAdd:
			logrus.Infof("add service: %+v", member)
		case EventDelete:
			logrus.Infof("delete service: %+v", member)
		case EventStatusChange:
			logrus.Infof("offline service: %+v", member)
		}
	})

	sev := NewService(client.Agent(), serviceName, "127.0.0.1", 7770)
	// event add
	sev.Register()
	// event status change
	// use updatettl will not trigger event chang
	time.Sleep(time.Second * 30)
	time.Sleep(time.Second)
	// event delete
	sev.Deregister()
	wg.Wait()
}