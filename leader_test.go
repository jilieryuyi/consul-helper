package consul

import (
	"testing"
	"github.com/sirupsen/logrus"
)

func TestNewLeader(t *testing.T) {
	leader := NewLeader(
		"127.0.0.1:8500",
		"test",
		"service-test",
		"127.0.0.1",
		7770,
	)
	leader.Select(func(member *ServiceMember) {
		logrus.Infof("%+v", member)
	})
}
