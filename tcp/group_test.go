package tcp

import "testing"

// go test -v -test.run Test_Clients
func Test_Clients(t *testing.T) {
	client := make(Clients, 0)
	n := new(ClientNode)
	client.append(n)

	if len(client) != 1 {
		t.Errorf("client error")
	}

	client.remove(n)

	if len(client) != 0 {
		t.Errorf("client error")
	}
}
