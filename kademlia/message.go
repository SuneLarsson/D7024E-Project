package kademlia

type MessageType string

const (
	PING       MessageType = "PING"
	PONG       MessageType = "PONG"
	STORE      MessageType = "STORE"
	FIND_NODE  MessageType = "FIND_NODE"
	FIND_VALUE MessageType = "FIND_VALUE"
)

type Message struct {
	Type    MessageType
	From    Contact
	To      Contact // Do i need to include the To field in the Ping message?
	Payload []byte
	RPCID   KademliaID // Unique ID for matching requests and responses
}

// type Ping struct {
// 	Message
// }

func NewPingMessage(from Contact, rpcID KademliaID, to Contact) *Message {
	return &Message{
		Type:  PING,
		From:  from,
		RPCID: rpcID,
		To:    to,
	}
}

func NewPongMessage(from Contact, rpcID KademliaID, to Contact) *Message {
	return &Message{
		Type:  PONG,
		From:  from,
		RPCID: rpcID,
		To:    to,
	}
}

func NewFindNodeMessage(from Contact, rpcID KademliaID, to Contact, target KademliaID) *Message {
	return &Message{
		Type:    FIND_NODE,
		From:    from,
		To:      to,
		Payload: []byte(target.String()),
		RPCID:   rpcID,
	}
}