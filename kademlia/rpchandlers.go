package kademlia

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
)

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
	case FIND_VALUE:
		kademlia.handleFindValue(msg)
	case FIND_VALUE_RESPONSE:
		kademlia.handleResponse(msg)
	default:
		fmt.Println("Unknown message:", msg.Type)
	}
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

func (kademlia *Kademlia) handlePing(msg Message) {
	fmt.Printf("Received PING from %s\n", msg.From.Address)
	pong := NewPongMessage(kademlia.Self, msg.RPCID, msg.From)
	kademlia.Network.SendMessage(msg.From.Address, pong)
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
	storeResult := true
	kademlia.DataStore.Put(key.String(), value)
	if err := recover(); err != nil {
		storeResult = false
	}

	msgResponse := NewStoreResponseMessage(kademlia.Self, msg.RPCID, msg.From, storeResult)
	kademlia.Network.SendMessage(msg.From.Address, msgResponse)

	// Send STORE_RESPONSE back to the sender
}

// Handle FIND_VALUE
func (kademlia *Kademlia) handleFindValue(msg Message) {
	fmt.Printf("Received FIND_VALUE from %s\n", &msg.From)
	targetID := &KademliaID{}
	err := json.Unmarshal(msg.Payload, targetID)
	if err != nil {
		fmt.Println("Error unmarshaling target ID:", err)
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
