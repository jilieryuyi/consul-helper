package consul

import (
	"testing"
	"time"
	"github.com/sirupsen/logrus"
)

func TestNewLeader(t *testing.T) {
	serviceName := "service-test"
	lockKey := "test"
	address := "127.0.0.1:8500"


	leader1 := NewLeader(
		address,
		lockKey,
		serviceName,
		"127.0.0.1",
		7770,
	)
	leader1.Free()
	leader1.Select(func(member *ServiceMember) {
		logrus.Infof("1=>%+v", member)
		// leader1 should be leader
		if !member.IsLeader {
			t.Errorf("select leader error")
		}
	})
	//defer leader1.Free()
	// wait a second, and start anther service
	time.Sleep(time.Second)

	leader2 := NewLeader(
		address,
		lockKey,
		serviceName,
		"127.0.0.1",
		7771,
	)
	leader2.Select(func(member *ServiceMember) {
		//leader2 should not be leader
		logrus.Infof("2=>%+v", member)
		if member.IsLeader {
			t.Errorf("select leader error")
		}
	})
	//defer leader2.Free()
	a := time.After(time.Second * 30)
	select {
	case <- a:
		return
	}
}

func TestLeader_Register(t *testing.T) {
	serviceName := "service-test"
	lockKey := "test"
	address := "127.0.0.1:8500"


	leader1 := NewLeader(
		address,
		lockKey,
		serviceName,
		"127.0.0.1",
		7770,
	)
	_, err := leader1.Register()
	if err != nil {
		t.Errorf("register err")
	}

	err = leader1.UpdateTtl()
	if err != nil {
		t.Errorf("UpdateTtl err")
	}

	err = leader1.Deregister()
	if err != nil {
		t.Errorf("Deregister err")
	}

	err = leader1.UpdateTtl()
	if err == nil {
		t.Errorf("UpdateTtl err")
	}
}
