package kademlia

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type NetworkAPI interface {
	Listen() error
	SendMessage(addr string, msg *Message) error
}

type Network struct {
	Self      Contact
	Conn      *net.UDPConn
	onMessage func(msg Message, addr *net.UDPAddr)
	workWg    sync.WaitGroup
}

func NewNetwork(self Contact, conn *net.UDPConn, handler func(msg Message, addr *net.UDPAddr)) *Network {
	return &Network{
		Self:      self,
		Conn:      conn,
		onMessage: handler,
		workWg:    sync.WaitGroup{},
	}
}

func (network *Network) Listen() error {
	// Create a UDP listener
	// defer network.Conn.Close()
	for {

		buffer := make([]byte, 20480)
		len, remoteAddr, err := network.Conn.ReadFromUDP(buffer)

		// log.Printf("DEBUG: Received %d bytes from %s", len, remoteAddr)

		if err != nil {
			// fmt.Println("Error reading from UDP:", err)
			continue
		}

		var msg Message
		if err := json.Unmarshal(buffer[:len], &msg); err != nil {
			fmt.Println("Error unmarshaling message:", err)
			continue
		}

		if network.onMessage != nil {
			network.workWg.Add(1)
			go func(msg Message, remoteAddr *net.UDPAddr) {
				defer network.workWg.Done()
				network.onMessage(msg, remoteAddr)
			}(msg, remoteAddr)
		}

	}
}

func (network *Network) SendMessage(addr string, msg *Message) error {

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
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

func (network *Network) Shutdown() error {
	error := network.Conn.Close()
	network.workWg.Wait()
	return error
}
