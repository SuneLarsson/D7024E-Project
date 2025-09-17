package kademlia

import (
	"encoding/json"
	"fmt"
	"net"
)

type NetworkAPI interface {
	Listen() error
	SendMessage(addr string, msg *Message) error
}

type Network struct {
	Self      Contact
	Conn      *net.UDPConn
	onMessage func(msg Message, addr *net.UDPAddr)
}

func NewNetwork(self Contact, conn *net.UDPConn, handler func(msg Message, addr *net.UDPAddr)) *Network {
	return &Network{
		Self:      self,
		Conn:      conn,
		onMessage: handler,
	}
}

func (network *Network) Listen() error {
	// Create a UDP listener
	defer network.Conn.Close()
	for {
		for {
			buffer := make([]byte, 2048)
			len, remoteAddr, err := network.Conn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Println("Error reading from UDP:", err)
				continue
			}

			var msg Message
			if err := json.Unmarshal(buffer[:len], &msg); err != nil {
				fmt.Println("Error unmarshaling message:", err)
				continue
			}

			if network.onMessage != nil {
				go network.onMessage(msg, remoteAddr)
			}
		}

	}
}

func (network *Network) SendMessage(addr string, msg *Message) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	data, err := json.Marshal(msg) //Kanske byt till protobuf
	if err != nil {
		fmt.Println("Error marshaling message:", err)
		return err
	}

	_, err = network.Conn.WriteToUDP(data, udpAddr)
	return err
}
