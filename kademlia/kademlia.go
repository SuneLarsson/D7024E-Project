package kademlia

import (
	"d7024e/storage"
	"fmt"
	"log"
	"net"
	"time"
)

// const kSize = 20 // Bucket size
// const alpha = 3  // Concurrency

// Kademlia structure

type Kademlia struct {
	Self         Contact
	Network      NetworkAPI
	RoutingTable *RoutingTable
	mapManagerCh chan MapRequest
	DataStore    storage.Storage
}

type MapRequest struct {
	rpcID        KademliaID
	responseChan chan Message
	register     bool
	responseMsg  Message
}

type DataItem struct {
	value      string
	timeToLive time.Time
}

func NewKademliaNode(ip string, port int) (*Kademlia, error) {
	// 1. Resolve the listening address (using "0.0.0.0" is correct here)
	listenAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", "0.0.0.0", port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return nil, err
	}

	outboundIP, err := getOutboundIP()
	if err != nil {
		log.Printf("Could not determine outbound IP, falling back to localhost: %v", err)
		outboundIP = "127.0.0.1" // Fallback for local testing
	}

	// 3. Create the Contact with the CORRECT, public address
	contact := Contact{
		ID:       NewRandomKademliaID(),
		Address:  fmt.Sprintf("%s:%d", outboundIP, port), // Use the discovered IP
		distance: nil,
	}

	routingtable := NewRoutingTable(contact)

	kademlia := &Kademlia{
		Self:         contact,
		RoutingTable: routingtable,
		mapManagerCh: make(chan MapRequest),
		DataStore:    *storage.NewStorage(),
	}

	network := NewNetwork(contact, conn, kademlia.HandleMessage)

	kademlia.Network = network

	go kademlia.Network.Listen()
	go kademlia.managePendingRequests()

	return kademlia, nil
}

func (k *Kademlia) managePendingRequests() {
	pending := make(map[string]chan Message)

	for req := range k.mapManagerCh {
		if req.register {
			pending[req.rpcID.String()] = req.responseChan
		} else {
			if ch, ok := pending[req.rpcID.String()]; ok {
				ch <- req.responseMsg
				delete(pending, req.rpcID.String())
			}
		}
	}
}

func (kademlia *Kademlia) JoinNetwork(knownContact *Contact) {
	//1. Create ID if not exists
	if kademlia.Self.ID == nil {
		kademlia.Self.ID = NewRandomKademliaID()
	}

	//2. Insert known contact into routing table (correct bucket)
	kademlia.RoutingTable.AddContact(*knownContact)

	//3. Run an Iterative Find Node on Self
	kademlia.IterativeFindNode(kademlia.Self.ID, 3, 20)

	//4. Refresh bucket further away than closest
	// neighbor
	closest := kademlia.RoutingTable.FindClosestContacts(kademlia.Self.ID, 1)
	bucketIndex := kademlia.RoutingTable.getBucketIndex(closest[0].ID)
	for i := bucketIndex + 1; i < IDLength*8; i++ {
		kademlia.RefreshBucket(i)
	}
}

func (kademlia *Kademlia) RefreshBucket(idx int) {
	contact := kademlia.RoutingTable.buckets[idx].getContactForBucketRefresh()
	if contact.ID != nil {
		kademlia.IterativeFindNode(contact.ID, 3, 20)
	}
}

// Helper function to get the outbound IP address
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
