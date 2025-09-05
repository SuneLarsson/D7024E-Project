package kademlia

import (
	"encoding/json"
	"fmt"
	"net"
)

type Network struct {
	Self     Contact
	Conn     *net.UDPConn
	Incoming chan Message
}

// type Message struct {
// }

func Listen(ip string, port int) (*Network, error) {
	// TODO
	// TODO
	// Create a UDP listener
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	self := Contact{
		ID:      NewRandomKademliaID(),
		Address: addr.String(),
	}

	network := &Network{
		Self:     self,
		Conn:     conn,
		Incoming: make(chan Message, 5),
	}
	go network.listenLoop()

	return network, nil
}

func (n *Network) listenLoop() {
	buf := make([]byte, 1024)
	for {
		nBytes, remoteAddr, err := n.Conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		var msg Message
		if err := json.Unmarshal(buf[:nBytes], &msg); err != nil {
			continue
		}

		fmt.Printf("Got message %s from %s\n", msg.Type, remoteAddr)

		// If it's a PING, reply with PONG
		if msg.Type == "PING" {
			pong := NewPongMessage(n.Self, msg.From)
			data, _ := json.Marshal(pong)
			udpAddr, _ := net.ResolveUDPAddr("udp", msg.From.Address)
			n.Conn.WriteToUDP(data, udpAddr)
			fmt.Printf("Sent PONG to %s\n", msg.From.Address)
		}

		n.Incoming <- msg
	}
}

func (network *Network) SendPingMessage(contact *Contact) {
	// TODO
}

func (network *Network) SendFindContactMessage(contact *Contact) {
	// TODO
}

func (network *Network) SendFindDataMessage(hash string) {
	// TODO
}

func (network *Network) SendStoreMessage(data []byte) {
	// TODO
}
