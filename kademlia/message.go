package kademlia

import "encoding/json"

// MessageType identifies the kind of Kademlia RPC being carried.
type MessageType string

// Supported Kademlia message types.
const (
	PING                MessageType = "PING"
	PONG                MessageType = "PONG"
	STORE               MessageType = "STORE"
	STORE_RESPONSE      MessageType = "STORE_RESPONSE"
	FIND_NODE_REQUEST   MessageType = "FIND_NODE_REQUEST"
	FIND_NODE_RESPONSE  MessageType = "FIND_NODE_RESPONSE"
	FIND_VALUE          MessageType = "FIND_VALUE"
	FIND_VALUE_RESPONSE MessageType = "FIND_VALUE_RESPONSE"
	REFRESH             MessageType = "REFRESH"
	REFRESH_RESPONSE    MessageType = "REFRESH_RESPONSE"
)

// Message carries a Kademlia RPC with metadata and an optional JSON-encoded
// payload. The payload schema depends on Type:
//   - FIND_NODE_REQUEST: KademliaID (target)
//   - FIND_NODE_RESPONSE: []Contact (closest contacts)
//   - STORE: string (value)
//   - STORE_RESPONSE: bool (store result)
//   - FIND_VALUE: KademliaID (key)
//   - FIND_VALUE_RESPONSE: string (value) or []Contact (closest contacts)
//   - REFRESH: KademliaID (key)
//   - REFRESH_RESPONSE: bool (refresh result)
//
// OriginalUploader indicates whether the sender is the original uploader of the
// value (affects republish behavior elsewhere). RPCID is used to correlate
// requests and their responses.
type Message struct {
	Type             MessageType
	From             Contact
	To               Contact
	Payload          []byte
	OriginalUploader bool
	RPCID            KademliaID // Unique ID for matching requests and responses
}

// NewPingMessage constructs a PING message without payload.
func NewPingMessage(from Contact, rpcID KademliaID, to Contact) *Message {
	return &Message{
		Type:  PING,
		From:  from,
		RPCID: rpcID,
		To:    to,
	}
}

// NewPongMessage constructs a PONG response without payload.
func NewPongMessage(from Contact, rpcID KademliaID, to Contact) *Message {
	return &Message{
		Type:  PONG,
		From:  from,
		RPCID: rpcID,
		To:    to,
	}
}

// NewFindNodeMessage constructs a FIND_NODE request containing the target ID.
func NewFindNodeMessage(from Contact, rpcID KademliaID, to Contact, target KademliaID) *Message {
	targetBytes, _ := json.Marshal(target)
	return &Message{
		Type:    FIND_NODE_REQUEST,
		From:    from,
		To:      to,
		Payload: targetBytes,
		RPCID:   rpcID,
	}
}

// ResponseFindNodeMessage constructs a FIND_NODE response carrying closest contacts.
func ResponseFindNodeMessage(from Contact, rpcID KademliaID, to Contact, contacts []Contact) *Message {
	contactsBytes, _ := json.Marshal(contacts)
	return &Message{
		Type:    FIND_NODE_RESPONSE,
		From:    from,
		To:      to,
		Payload: contactsBytes,
		RPCID:   rpcID,
	}
}

// NewStoreMessage constructs a STORE request carrying the value to be stored.
// The key is derived by receivers as SHA-1(value). originalUploader marks the
// sender as the original uploader for republish decisions elsewhere.
func NewStoreMessage(from Contact, rpcID KademliaID, to Contact, data string, originalUploader bool) *Message {
	dataBytes, _ := json.Marshal(data)
	return &Message{
		Type:             STORE,
		From:             from,
		To:               to,
		Payload:          dataBytes,
		OriginalUploader: originalUploader,
		RPCID:            rpcID,
	}
}

// NewStoreResponseMessage constructs a STORE_RESPONSE carrying the boolean result.
func NewStoreResponseMessage(from Contact, rpcID KademliaID, to Contact, result bool) *Message {
	resultBytes, _ := json.Marshal(result)
	return &Message{
		Type:    STORE_RESPONSE,
		From:    from,
		To:      to,
		Payload: resultBytes,
		RPCID:   rpcID,
	}
}

// NewFindValueMessage constructs a FIND_VALUE request for the given key ID.
func NewFindValueMessage(from Contact, rpcID KademliaID, to Contact, key KademliaID) *Message {
	keyBytes, _ := json.Marshal(key)
	return &Message{
		Type:    FIND_VALUE,
		From:    from,
		To:      to,
		Payload: keyBytes,
		RPCID:   rpcID,
	}
}

// NewFindValueResponseMessage constructs a FIND_VALUE response. If value is
// non-empty, it is returned as the payload; otherwise, the payload carries the
// list of closest contacts.
func NewFindValueResponseMessage(from Contact, rpcID KademliaID, to Contact, value string, contacts []Contact) *Message {
	var payload []byte
	if value != "" {
		payload, _ = json.Marshal(value)
	} else {
		contactsBytes, _ := json.Marshal(contacts)
		payload = contactsBytes
	}
	return &Message{
		Type:    FIND_VALUE_RESPONSE,
		From:    from,
		To:      to,
		Payload: payload,
		RPCID:   rpcID,
	}
}

// NewRefreshMessage constructs a REFRESH request for the given key ID.
func NewRefreshMessage(from Contact, rpcID KademliaID, to Contact, key KademliaID) *Message {
	keyBytes, _ := json.Marshal(key)
	return &Message{
		Type:    REFRESH,
		From:    from,
		To:      to,
		Payload: keyBytes,
		RPCID:   rpcID,
	}
}

// NewRefreshResponseMessage constructs a REFRESH_RESPONSE carrying the boolean result.
func NewRefreshResponseMessage(from Contact, rpcID KademliaID, to Contact, result bool) *Message {
	resultBytes, _ := json.Marshal(result)
	return &Message{
		Type:    REFRESH_RESPONSE,
		From:    from,
		To:      to,
		Payload: resultBytes,
		RPCID:   rpcID,
	}
}
