package kademlia

import (
	"fmt"
	"net"
	"time"
)

const kSize = 20 // Bucket size
const alpha = 3  // Concurrency

// Kademlia structure

type Kademlia struct {
	Self         Contact
	Network      NetworkAPI
	RoutingTable *RoutingTable
	mapManagerCh chan MapRequest
	DataStore    map[KademliaID]DataItem
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

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	contact := Contact{
		ID:       NewRandomKademliaID(),
		Address:  addr.String(),
		distance: nil,
	}

	routingtable := NewRoutingTable(contact)

	kademlia := &Kademlia{
		Self:         contact,
		RoutingTable: routingtable,
		mapManagerCh: make(chan MapRequest),
		DataStore:    make(map[KademliaID]DataItem),
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
	kademlia.IterativeFindNode(kademlia.Self.ID)

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
		kademlia.IterativeFindNode(contact.ID)
	}
}

// Our implementation of LookupNode
