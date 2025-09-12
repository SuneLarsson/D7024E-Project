package kademlia

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Kademlia struct {
	Self         Contact
	Network      *Network
	RoutingTable *RoutingTable
	mapManagerCh chan MapRequest
}

type MapRequest struct {
	rpcID        KademliaID
	responseChan chan Message
	register     bool
	responseMsg  Message
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

func (kademlia *Kademlia) LookupContact(target *Contact) {
	// TODO
}

func (kademlia *Kademlia) LookupData(hash string) {
	// TODO
}

func (kademlia *Kademlia) Store(data []byte) {
	// TODO
}

// func (kademlia *Kademlia) JoinNetwork(network *Network, knownContact *Contact) {
// 	//1. Create ID if not exists
// 	if network.Self.ID == nil {
// 		network.Self.ID = NewRandomKademliaID()
// 	}

// 	//2. Insert known contact into routing table (correct bucket)
// 	kademlia.Network.RoutingTable.AddContact(*knownContact)

// 	//3. Run an Iterative Find Node on Self
// 	kademlia.Network.IterativeFindNode(kademlia.Self.ID)

// 	closest := kademlia.RoutingTable.FindClosestContacts(kademlia.Self.ID, 1)

// 	//4. Refresh bucket further away than closest
// 	// neighbor
// 	kademlia.Network.RoutingTable.RefreshBuckets()
// }

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

func (kademlia *Kademlia) IterativeFindNode(target *KademliaID) []Contact {

	//1. Start with closest known contacts
	//2. Ask a few nodes at a time (alpha)
	//3. Merge responses, only keep closest contacts
	//4. Repeat until no new nodes are found
	const kSize = 20 // GOAL
	const alpha = 3  // Concurrency
	candidates := &ContactCandidates{}
	shortlist := kademlia.RoutingTable.FindClosestContacts(target, alpha)
	candidates.Append(shortlist)

	queried := make(map[string]bool)
	// results := make([]Contact, 0, kSize)

	for {
		nodesToQuery := &ContactCandidates{}
		for _, c := range shortlist {
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
	// Handle STORE message
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
