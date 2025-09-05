package kademlia

import (
	"testing"
	"time"
)

func TestPingPong(t *testing.T) {

	node1 := NewNode("127.0.0.1", 8001)
	node2 := NewNode("127.0.0.1", 8002)

	contact2 := &Contact{
		ID:      node2.ID,
		Address: node2.Address,
	}
	node1.Ping(contact2)

	select {
	case msg := <-node1.Network.Incoming:
		if msg.Type != "PONG" {
			t.Errorf("expected PONG, got %s", msg.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout: node1 did not receive PONG")
	}
}
