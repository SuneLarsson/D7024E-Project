package kademlia

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

func (kademlia *Kademlia) SendPing(contact *Contact) error {
	rpcID := NewRandomKademliaID()

	responseChan := make(chan Message, 1)

	req := MapRequest{
		rpcID:        *rpcID,
		responseChan: responseChan,
		register:     true,
	}

	kademlia.mapManagerCh <- req

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    *rpcID,
			register: false,
		}
		kademlia.mapManagerCh <- deregisterReq
	}()

	pingMsg := NewPingMessage(kademlia.Self, *rpcID, *contact)

	fmt.Printf("PING message: %+v\n", pingMsg)
	fmt.Printf("Sending PING to %s \n", contact.Address)

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

func (kademlia *Kademlia) FindNode(contact *Contact, target *KademliaID) ([]Contact, bool, *string) {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	kademlia.mapManagerCh <- req

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    rpcID,
			register: false,
		}
		kademlia.mapManagerCh <- deregisterReq
	}()

	findMsg := NewFindNodeMessage(kademlia.Self, rpcID, *contact, *target)
	err := kademlia.Network.SendMessage(contact.Address, findMsg)
	if err != nil {
		fmt.Println("Error sending FindNode message:", err)
		return nil, false, nil
	}

	select {
	case resp := <-req.responseChan:
		if resp.Type == FIND_NODE_RESPONSE {
			var contacts []Contact
			if err := json.Unmarshal(resp.Payload, &contacts); err != nil {
				fmt.Println("Error unmarshaling contacts:", err)
				return nil, false, nil
			}
			return contacts, true, nil
		}
	case <-time.After(3 * time.Second):
		// Timeout
		//TODO add a breakout
		fmt.Println("FindNode request timed out")
	}
	return []Contact{}, false, nil
}

// STORE
// The sender of the STORE RPC provides a key and a block of data and requires that the recipient store the data and make it available for later retrieval by that key.

// This is a primitive operation, not an iterative one.
func (kademlia *Kademlia) Store(contact *Contact, value string, hash string, originalUploader bool) bool {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	kademlia.mapManagerCh <- req

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    rpcID,
			register: false,
		}
		kademlia.mapManagerCh <- deregisterReq
	}()

	storeMsg := NewStoreMessage(kademlia.Self, rpcID, *contact, value, originalUploader)
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

// FIND_VALUE
func (kademlia *Kademlia) FindValue(contact *Contact, target *KademliaID) ([]Contact, bool, *string) {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	kademlia.mapManagerCh <- req

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    rpcID,
			register: false,
		}
		kademlia.mapManagerCh <- deregisterReq
	}()

	findValueMsg := NewFindValueMessage(kademlia.Self, rpcID, *contact, *target)
	err := kademlia.Network.SendMessage(contact.Address, findValueMsg)
	if err != nil {
		fmt.Println("Error sending FindValue message:", err)
		return nil, false, nil
	}

	select {
	case resp := <-req.responseChan:
		if resp.Type == FIND_VALUE_RESPONSE {
			var value *string
			if err := json.Unmarshal(resp.Payload, &value); err == nil {
				// Case 1: Node had a value
				if value != nil {
					return nil, true, value
				}
				// Case 2: Node explicitly had no value (nil string)
				return nil, false, nil
			}

			// Case 3: Try contacts instead
			var contacts []Contact
			if err := json.Unmarshal(resp.Payload, &contacts); err == nil {
				return contacts, false, nil
			}

			fmt.Println("Error unmarshaling FIND_VALUE_RESPONSE:", err)
			return nil, false, nil
		}
	case <-time.After(3 * time.Second):
		// Timeout
		fmt.Println("FindValue request timed out")
	}
	return nil, false, nil
}

func (kademlia *Kademlia) Refresh(contact *Contact, key string) bool {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	kademlia.mapManagerCh <- req

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    rpcID,
			register: false,
		}
		kademlia.mapManagerCh <- deregisterReq
	}()

	refreshMsg := NewRefreshMessage(kademlia.Self, rpcID, *contact, key)
	err := kademlia.Network.SendMessage(contact.Address, refreshMsg)
	if err != nil {
		fmt.Println("Error sending REFRESH message:", err)
	}
	select {
	case resp := <-req.responseChan:
		if resp.Type == REFRESH_RESPONSE {
			var result bool
			if err := json.Unmarshal(resp.Payload, &result); err != nil {
				fmt.Println("Error unmarshaling result:", err)
				return false
			}
			return result
		}
	case <-time.After(3 * time.Second):
		// Timeout
		fmt.Println("Refresh request timed out")
		return false
	}
	return false
}

// FORGET VALUE/STOP REFRESHING
func (kademlia *Kademlia) Forget(key string) {
	kademlia.keyMutex.Lock()
	defer kademlia.keyMutex.Unlock()
	if kademlia.keyStore[key] != nil {
		kademlia.keyStore[key] <- "Forget"
		delete(kademlia.keyStore, key)
	}
}
