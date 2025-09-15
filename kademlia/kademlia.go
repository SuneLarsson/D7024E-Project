package kademlia

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const kSize = 20 // Bucket size
const alpha = 3  // Concurrency

// Kademlia structure

type Kademlia struct {
	Self         Contact
	Network      *Network
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

	network := &Network{
		Conn: conn,
		Self: contact,
	}

	routingtable := NewRoutingTable(contact)

	kademlia := &Kademlia{
		Self:         contact,
		RoutingTable: routingtable,
		Network:      network,
		mapManagerCh: make(chan MapRequest),
	}

	kademlia.Network.onMessage = kademlia.HandleMessage

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

func (kademlia *Kademlia) HandleMessage(msg Message, addr *net.UDPAddr) {
	kademlia.RoutingTable.AddContact(msg.From)

	switch msg.Type {
	case PING:
		kademlia.handlePing(msg)
	case PONG:
		kademlia.handleResponse(msg)
	case FIND_NODE_REQUEST:
		kademlia.handleFindNode(msg)
	case FIND_NODE_RESPONSE:
		kademlia.handleResponse(msg)
	case STORE:
		kademlia.handleStore(msg)
	case STORE_RESPONSE:
		kademlia.handleResponse(msg)
	default:
		fmt.Println("Unknown message:", msg.Type)
	}
}

func (kademlia *Kademlia) handlePing(msg Message) {
	fmt.Printf("Received PING from %s\n", msg.From.Address)
	pong := NewPongMessage(kademlia.Self, msg.RPCID, msg.From)
	kademlia.Network.SendMessage(msg.From.Address, pong)
}

// General dispatch for responses that are "final"
func (k *Kademlia) handleResponse(msg Message) {
	dispatchRequest := MapRequest{
		rpcID:       msg.RPCID,
		responseMsg: msg,
		register:    false,
	}

	k.mapManagerCh <- dispatchRequest
}

func (kademlia *Kademlia) handleFindNode(msg Message) {
	fmt.Printf("Received FIND_NODE from %s\n", &msg.From)
	targetID := &KademliaID{}
	err := json.Unmarshal(msg.Payload, targetID)
	if err != nil {
		fmt.Println("Error unmarshaling target ID:", err)
		return
	}

	closest := kademlia.RoutingTable.FindClosestContacts(targetID, bucketSize)
	response := ResponseFindNodeMessage(kademlia.Self, msg.RPCID, msg.From, closest)
	kademlia.Network.SendMessage(msg.From.Address, response)
}

func (kademlia *Kademlia) LookupData(hash string) {

}

// STORE
// The sender of the STORE RPC provides a key and a block of data and requires that the recipient store the data and make it available for later retrieval by that key.

// This is a primitive operation, not an iterative one.
func (kademlia *Kademlia) Store(contact *Contact, value string, hash string) bool {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	kademlia.mapManagerCh <- req

	storeMsg := NewStoreMessage(kademlia.Self, rpcID, *contact, value)
	err := kademlia.Network.SendMessage(contact.Address, storeMsg)
	if err != nil {
		fmt.Println("Error sending STORE message:", err)
	}

	select {
	case resp := <-req.responseChan:
		if resp.Type == STORE_RESPONSE {
			var result bool
			if err := json.Unmarshal(resp.Payload, &result); err != nil {
				fmt.Println("Error unmarshaling result:", err)
				return false
			}
			return result
		}
	case <-time.After(3 * time.Second):
		// Timeout
		fmt.Println("Store request timed out")
		return false
	}
	return false
}

func (kademlia *Kademlia) iterativeStore(value string) {
	//1. Hash the value to get the key
	dataToHash := []byte(value)
	hash := sha1.Sum(dataToHash)
	key := NewKademliaID(hex.EncodeToString(hash[:]))

	//2. Find the k closest nodes to the key
	closest := kademlia.IterativeFindNode(key)
	//3. Send STORE RPCs to those nodes
	successCount := 0

	for _, contact := range closest {
		result := kademlia.Store(&contact, value, key.String())
		if result {
			successCount++
		}
	}
	// If at least one STORE was successful, consider it a success
	// and print the number of successful stores
	// Otherwise, print a failure message
	if successCount > 0 {
		fmt.Printf("Successfully stored value on %d nodes\n", successCount)
	} else {
		fmt.Println("Failed to store value on any node")
	}

	//4. If a node does not respond, find a replacement node and send STORE to it // Optional

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

func (kademlia *Kademlia) FindNode(contact *Contact, target *KademliaID) []Contact {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	kademlia.mapManagerCh <- req

	findMsg := NewFindNodeMessage(kademlia.Self, rpcID, *contact, *target)
	err := kademlia.Network.SendMessage(contact.Address, findMsg)
	if err != nil {
		fmt.Println("Error sending FindNode message:", err)
		return nil
	}

	select {
	case resp := <-req.responseChan:
		if resp.Type == FIND_NODE_RESPONSE {
			var contacts []Contact
			if err := json.Unmarshal(resp.Payload, &contacts); err != nil {
				fmt.Println("Error unmarshaling contacts:", err)
				return nil
			}
			return contacts
		}
	case <-time.After(3 * time.Second):
		// Timeout
		//TODO add a breakout
		fmt.Println("FindNode request timed out")
	}
	return []Contact{}
}

// Our implementation of LookupNode
func (kademlia *Kademlia) IterativeFindNode(target *KademliaID) []Contact {

	//1. Start with closest known contacts
	//2. Ask a few nodes at a time (alpha)
	//3. Merge responses, only keep closest contacts
	//4. Repeat until no new nodes are found

	candidates := &ContactCandidates{}
	shortlist := kademlia.RoutingTable.FindClosestContacts(target, alpha)
	candidates.Append(shortlist)

	queried := make(map[string]bool)
	// results := make([]Contact, 0, kSize)

	for {
		nodesToQuery := &ContactCandidates{}
		for _, c := range candidates.contacts {
			if nodesToQuery.Len() >= alpha {
				break
			}
			if !queried[c.ID.String()] {
				nodesToQuery.Append([]Contact{c})
			}
		}
		if nodesToQuery.Len() == 0 {
			break
		}

		responseChan := make(chan []Contact, nodesToQuery.Len())
		for _, c := range nodesToQuery.contacts {
			queried[c.ID.String()] = true
			go func(contact Contact) {
				res := kademlia.FindNode(&contact, target)
				if res != nil {
					responseChan <- res
				} else {
					responseChan <- []Contact{}
				}
			}(c)
		}

		for i := 0; i < nodesToQuery.Len(); i++ {
			newContacts := <-responseChan
			for _, nc := range newContacts {
				nc.CalcDistance(target)
				if !containsContact(candidates.contacts, nc) {
					candidates.Append([]Contact{nc})
				}
			}
		}

		candidates.Sort()
		if candidates.Len() > kSize {
			candidates.contacts = candidates.GetContacts(kSize)
		}
	}

	return candidates.contacts
}

func containsContact(list []Contact, c Contact) bool {
	for _, x := range list {
		if x.ID.Equals(c.ID) {
			return true
		}
	}
	return false
}

func (kademlia *Kademlia) handleFindValue(msg Message, addr *net.UDPAddr) {
	// Handle FIND_VALUE message
}

func (kademlia *Kademlia) handleStore(msg Message) {
	fmt.Printf("Received STORE from %s\n", &msg.From)
	var value string
	err := json.Unmarshal(msg.Payload, &value)
	if err != nil {
		fmt.Println("Error unmarshaling value:", err)
		return
	}
	hash := sha1.Sum([]byte(value))
	key := NewKademliaID(hex.EncodeToString(hash[:]))

	dataItem, err := kademlia.NewDataItem(value)
	storeResult := true
	if err != nil {
		fmt.Println("Error creating DataItem:", err)
		storeResult = false
	}
	kademlia.DataStore[*key] = *dataItem

	msgResponse := NewStoreResponseMessage(kademlia.Self, msg.RPCID, msg.From, storeResult)
	kademlia.Network.SendMessage(msg.From.Address, msgResponse)

	// Send STORE_RESPONSE back to the sender
}

func (kademlia *Kademlia) SendPing(contact *Contact) error {
	rpcID := NewRandomKademliaID()

	responseChan := make(chan Message, 1)

	req := MapRequest{
		rpcID:        *rpcID,
		responseChan: responseChan,
		register:     true,
	}

	kademlia.mapManagerCh <- req

	pingMsg := NewPingMessage(kademlia.Self, *rpcID, *contact)

	err := kademlia.Network.SendMessage(contact.Address, pingMsg)
	if err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}

	select {
	case pongMsg := <-responseChan:
		fmt.Printf("Received PONG from %s with ID %s\n", contact.Address, hex.EncodeToString(pongMsg.RPCID[:]))
		return nil

	case <-time.After(3 * time.Second):
		return fmt.Errorf("ping to %s timed out", contact.Address)
	}
}
