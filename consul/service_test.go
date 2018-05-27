package consul

import (
	"testing"
	"github.com/hashicorp/consul/api"
)

func TestNewService(t *testing.T) {
	address := "127.0.0.1:8500"
	config := api.DefaultConfig()
	config.Address = address
	client, _ := api.NewClient(config)
	agent := client.Agent()

	NewService(
		agent,
		"test",
		"127.0.0.1",
		7000)
	NewService(
		agent,
		"test",
		"127.0.0.1",
		7001)
	 NewService(
		agent,
		 "test",
		"127.0.0.1",
		7002,
		)
}
