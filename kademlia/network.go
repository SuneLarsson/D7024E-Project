package kademlia

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

// NetworkAPI abstracts the transport layer used by Kademlia.
// Implementations must be able to listen for incoming messages and send
// messages to a given remote address.
type NetworkAPI interface {
	Listen() error
	SendMessage(addr string, msg *Message) error
}

// Network is a UDP-based transport that encodes/decodes Message values as JSON.
// It invokes the provided onMessage handler for each received message in its
// own goroutine and uses a WaitGroup to track in-flight handlers for shutdown.
type Network struct {
	Self      Contact
	Conn      *net.UDPConn
	onMessage func(msg Message, addr *net.UDPAddr)
	workWg    sync.WaitGroup
}

// NewNetwork constructs a Network bound to the given UDP connection and
// installs the provided message handler.
func NewNetwork(self Contact, conn *net.UDPConn, handler func(msg Message, addr *net.UDPAddr)) *Network {
	return &Network{
		Self:      self,
		Conn:      conn,
		onMessage: handler,
		workWg:    sync.WaitGroup{},
	}
}

// Listen reads datagrams from the UDP connection, decodes each as a Message,
// and dispatches it to the onMessage handler in a separate goroutine. This call
// blocks until the underlying connection is closed. Errors on read or decode
// are logged and the loop continues.
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

// SendMessage resolves the remote UDP address, JSON-encodes msg, and sends it
// via the underlying UDP connection.
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

// Shutdown closes the UDP connection and waits for all in-flight message
// handlers to complete before returning.
func (network *Network) Shutdown() error {
	error := network.Conn.Close()
	network.workWg.Wait()
	return error
}
