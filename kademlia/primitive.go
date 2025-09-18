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

	pingMsg := NewPingMessage(kademlia.Self, *rpcID, *contact)

	fmt.Printf("PING message: %+v\n", pingMsg)

	err := kademlia.Network.SendMessage(contact.Address, pingMsg)
	if err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}

	select {
	case pongMsg := <-responseChan:
		fmt.Printf("Received PONG from %s with ID %s\n", contact.Address, hex.EncodeToString(pongMsg.RPCID[:]))
		return nil

	case <-time.After(10 * time.Second):
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

// FIND_VALUE
func (kademlia *Kademlia) FindValue(contact *Contact, target *KademliaID) ([]Contact, bool, *string) {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	kademlia.mapManagerCh <- req

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
			var contacts []Contact
			if err := json.Unmarshal(resp.Payload, &value); err != nil {
				// If unmarshaling to string fails, try unmarshaling to contacts
				if err := json.Unmarshal(resp.Payload, &contacts); err != nil {
					fmt.Println("Error unmarshaling value or contacts:", err)
					return nil, false, nil
				}
				// If we got contacts, return them with a false flag
				return contacts, false, nil
			}
			// If we got a value, return it with a true flag
			if value == nil {
				return contacts, false, nil // No value found, return contacts
			}
			return nil, true, value
		}
	case <-time.After(3 * time.Second):
		// Timeout
		fmt.Println("FindValue request timed out")
	}
	return nil, false, nil
}
