package kademlia

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
)

// HandleMessage updates the sender's contact information and dispatches the
// incoming RPC message to the appropriate handler. Responses that are final are
// forwarded to waiting goroutines via handleResponse, while requests are
// processed and responded to immediately.
func (kademlia *Kademlia) HandleMessage(msg Message, addr *net.UDPAddr) {
	// Update the sender's address in the Contact if the UDP address is known.
	if addr != nil {
		msg.From.Address = addr.String()
	}

	// Opportunistically add/update the sender in the routing table.
	go kademlia.RoutingTable.AddContact(msg.From)

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
	case FIND_VALUE:
		kademlia.handleFindValue(msg)
	case FIND_VALUE_RESPONSE:
		kademlia.handleResponse(msg)
	case REFRESH:
		kademlia.handleRefresh(msg)
	case REFRESH_RESPONSE:
		kademlia.handleResponse(msg)
	default:
		fmt.Println("Unknown message:", msg.Type)
	}
}

// handleResponse dispatches a terminal response message to the goroutine
// waiting on the corresponding RPCID.
func (k *Kademlia) handleResponse(msg Message) {
	// fmt.Printf("Received response of type %s from %s\n", msg.Type, msg.From.Address)
	dispatchRequest := MapRequest{
		rpcID:       msg.RPCID,
		responseMsg: msg,
		register:    false,
	}

	addMapRequest(k, dispatchRequest)
}

// handlePing replies to a PING with a PONG directed at the sender.
func (kademlia *Kademlia) handlePing(msg Message) {
	// fmt.Printf("Received PING from %s\n", msg.From.Address)
	pong := NewPongMessage(kademlia.Self, msg.RPCID, msg.From)
	kademlia.Network.SendMessage(msg.From.Address, pong)
}

// handleRefresh attempts to refresh the TTL of the given key if present in the
// local datapool by re-putting it. Responds with a boolean indicating whether
// the refresh was performed.
func (kademlia *Kademlia) handleRefresh(msg Message) {
	key := &KademliaID{}

	err := json.Unmarshal(msg.Payload, &key)
	if err != nil {
		// fmt.Println("Error unmarshaling key:", err)
		return
	}
	stored, exists := kademlia.DataStore.Get(key.String())
	refreshResult := false
	if exists {
		kademlia.DataStore.Put(key.String(), stored, false, true) // Refresh by re-putting
		refreshResult = true
	}
	response := NewRefreshResponseMessage(kademlia.Self, msg.RPCID, msg.From, refreshResult)
	kademlia.Network.SendMessage(msg.From.Address, response)
}

// handleStore stores the provided value locally, deriving the key as the SHA-1
// hash of the value. A STORE_RESPONSE with the success flag is sent back to the
// requester.
func (kademlia *Kademlia) handleStore(msg Message) {
	// fmt.Printf("Received STORE from %s\n", &msg.From)
	var value string
	err := json.Unmarshal(msg.Payload, &value)
	if err != nil {
		// fmt.Println("Error unmarshaling value:", err)
		return
	}
	// originalUploader := msg.OriginalUploader
	hash := sha1.Sum([]byte(value))
	key := NewKademliaID(hex.EncodeToString(hash[:]))
	storeResult := true
	kademlia.DataStore.Put(key.String(), value, false, msg.OriginalUploader)
	if err := recover(); err != nil {
		storeResult = false
	}

	msgResponse := NewStoreResponseMessage(kademlia.Self, msg.RPCID, msg.From, storeResult)
	kademlia.Network.SendMessage(msg.From.Address, msgResponse)

	// Send STORE_RESPONSE back to the sender
}

// handleFindValue serves a FIND_VALUE request. If the target key is present
// locally, the value is returned; otherwise, a list of closest contacts is
// returned.
func (kademlia *Kademlia) handleFindValue(msg Message) {
	// fmt.Printf("Received FIND_VALUE from %s\n", &msg.From)
	targetID := &KademliaID{}
	err := json.Unmarshal(msg.Payload, targetID)
	if err != nil {
		// fmt.Println("Error unmarshaling target ID:", err)
		return
	}

	dataItem, exists := kademlia.DataStore.Get(targetID.String())
	//lookup
	if exists {
		response := NewFindValueResponseMessage(kademlia.Self, msg.RPCID, msg.From, dataItem, nil)
		kademlia.Network.SendMessage(msg.From.Address, response)
		return
	} else {
		closest := kademlia.RoutingTable.FindClosestContacts(targetID, bucketSize)
		response := NewFindValueResponseMessage(kademlia.Self, msg.RPCID, msg.From, "", closest)
		kademlia.Network.SendMessage(msg.From.Address, response)
		return
	}

}

// handleFindNode serves a FIND_NODE request by returning the closest contacts
// to the requested target ID.
func (kademlia *Kademlia) handleFindNode(msg Message) {
	// fmt.Printf("Received FIND_NODE from %s\n", &msg.From)
	targetID := &KademliaID{}
	err := json.Unmarshal(msg.Payload, targetID)
	if err != nil {
		// fmt.Println("Error unmarshaling target ID:", err)
		return
	}

	closest := kademlia.RoutingTable.FindClosestContacts(targetID, bucketSize)
	response := ResponseFindNodeMessage(kademlia.Self, msg.RPCID, msg.From, closest)
	kademlia.Network.SendMessage(msg.From.Address, response)
}
