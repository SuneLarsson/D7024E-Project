package kademlia

import (
	"encoding/json"
	"fmt"
	"net"
)

type Network struct {
	Self      Contact
	Conn      *net.UDPConn
	onMessage func(msg Message, addr *net.UDPAddr)
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


func (network *Network) SendMessage(contact *Contact, msg *Message) error {
	udpAddr, err := net.ResolveUDPAddr("udp", contact.Address)
	if err != nil {
		return err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error marshaling message:", err)
		return err
	}

	_, err = network.Conn.WriteToUDP(data, udpAddr)
	return err
}

func (network *Network) SendFindDataMessage(hash string) {
	// TODO
}

func (network *Network) SendStoreMessage(data []byte) {
	// TODO
}
