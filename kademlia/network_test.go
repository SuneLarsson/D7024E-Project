package kademlia

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessageAndListen(t *testing.T) {
	addr1, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9101")
	conn1, _ := net.ListenUDP("udp", addr1)
	defer conn1.Close()

	addr2, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9102")
	conn2, _ := net.ListenUDP("udp", addr2)
	defer conn2.Close()

	var got Message
	done := make(chan struct{})

	handler := func(msg Message, from *net.UDPAddr) {
		got = msg
		close(done)
	}

	contact1 := NewContact(NewRandomKademliaID(), addr1.String())
	network1 := NewNetwork(contact1, conn1, handler)

	contact2 := NewContact(NewRandomKademliaID(), addr2.String())
	network2 := NewNetwork(contact2, conn2, nil)

	// Start listening on network1
	go network1.Listen()

	// Prepare message
	msg := NewPingMessage(contact2, *NewRandomKademliaID(), contact1)

	// Send from network2 → network1
	err := network2.SendMessage(addr1.String(), msg)
	require.NoError(t, err, "SendMessage should succeed")

	select {
	case <-done:
		assert.Equal(t, PING, got.Type, "Handler should receive PING message")
		assert.Equal(t, contact2.Address, got.From.Address, "Sender should match")
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for message to be received")
	}
}

func TestSendMessageInvalidAddr(t *testing.T) {
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9103")
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	contact := NewContact(NewRandomKademliaID(), addr.String())
	network := NewNetwork(contact, conn, nil)

	msg := NewPingMessage(contact, *NewRandomKademliaID(), contact)

	// Invalid address should error
	err := network.SendMessage("not_a_valid_addr", msg)
	assert.Error(t, err, "Invalid UDP address should return error")
}

func TestSendMessageMarshalError(t *testing.T) {
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9104")
	conn, _ := net.ListenUDP("udp", addr)
	contact := NewContact(NewRandomKademliaID(), addr.String())
	network := NewNetwork(contact, conn, nil)

	conn.Close() // closed connection should trigger error

	msg := NewPingMessage(contact, *NewRandomKademliaID(), contact)
	err := network.SendMessage(addr.String(), msg)
	assert.Error(t, err, "Closed connection should cause SendMessage error")
}
