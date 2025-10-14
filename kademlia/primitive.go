package kademlia

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// SendPing sends a PING RPC to the given contact and waits for a PONG
// response. It registers a temporary MapRequest so the asynchronous response
// can be routed back to this call. Returns an error on send failure or if the
// 3-second response timeout elapses.
func (kademlia *Kademlia) SendPing(contact *Contact) error {
	rpcID := NewRandomKademliaID()

	responseChan := make(chan Message, 1)

	req := MapRequest{
		rpcID:        *rpcID,
		responseChan: responseChan,
		register:     true,
	}
	addMapRequest(kademlia, req)

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    *rpcID,
			register: false,
		}
		addMapRequest(kademlia, deregisterReq)
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

// FindNode sends a FIND_NODE RPC to contact for the specified target ID and
// waits for a response up to 3 seconds. It returns the list of contacts
// returned by the peer, a boolean indicating success, and a third value which
// is always nil (kept for a uniform signature across find operations).
func (kademlia *Kademlia) FindNode(contact *Contact, target *KademliaID) ([]Contact, bool, *string) {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	addMapRequest(kademlia, req)

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    rpcID,
			register: false,
		}
		addMapRequest(kademlia, deregisterReq)
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
		// fmt.Println("FindNode request timed out")
	}
	return []Contact{}, false, nil
}

// addMapRequest enqueues a MapRequest to the manager goroutine so that
// responses can be routed back to the initiating call.
func addMapRequest(kademlia *Kademlia, req MapRequest) {
	kademlia.mapManagerCh <- req
}

// Store sends a STORE RPC to contact with the given value. The key used by the
// remote is the SHA-1 of the value (computed by the receiver as well). The
// originalUploader flag signals whether the caller is the original uploader,
// affecting refresh/republish semantics elsewhere. Returns true if the peer
// acknowledged storing the value; times out after 3 seconds.
//
// This is a primitive one-shot RPC, not an iterative procedure.
func (kademlia *Kademlia) Store(contact *Contact, value string, hash string, originalUploader bool) bool {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	addMapRequest(kademlia, req)

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    rpcID,
			register: false,
		}
		addMapRequest(kademlia, deregisterReq)
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

// FindValue sends a FIND_VALUE RPC to contact for the given target key. It
// returns one of the following cases:
//   - (nil, true, value) if the peer had the value
//   - (contacts, false, nil) if the peer returns closer contacts instead
//   - (nil, false, nil) on timeout or parse error
//
// The call waits up to 3 seconds for a response.
func (kademlia *Kademlia) FindValue(contact *Contact, target *KademliaID) ([]Contact, bool, *string) {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	addMapRequest(kademlia, req)

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    rpcID,
			register: false,
		}
		addMapRequest(kademlia, deregisterReq)
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

// Refresh sends a REFRESH RPC to contact for the given key to extend its local
// TTL if present. Returns true if the peer refreshed the value; waits up to 3
// seconds for a response.
func (kademlia *Kademlia) Refresh(contact *Contact, key string) bool {
	rpcID := *NewRandomKademliaID()

	req := MapRequest{
		rpcID:        rpcID,
		responseChan: make(chan Message, 1),
		register:     true,
	}
	addMapRequest(kademlia, req)

	defer func() {
		deregisterReq := MapRequest{
			rpcID:    rpcID,
			register: false,
		}
		addMapRequest(kademlia, deregisterReq)
	}()

	refreshMsg := NewRefreshMessage(kademlia.Self, rpcID, *contact, *NewKademliaID(key))
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
		// fmt.Println("Refresh request timed out")
		return false
	}
	return false
}

// Forget stops periodic refresh/republish for the given key if it is currently
// scheduled, by signaling the associated channel and removing it from the
// keyStore.
func (kademlia *Kademlia) Forget(key string) {
	kademlia.keyMutex.Lock()
	defer kademlia.keyMutex.Unlock()
	if kademlia.keyStore[key] != nil {
		kademlia.keyStore[key] <- "Forget"
		delete(kademlia.keyStore, key)
	}
}
