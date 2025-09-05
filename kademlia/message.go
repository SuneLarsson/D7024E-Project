package kademlia

type MessageType string

const (
	PING MessageType = "PING"
	PONG MessageType = "PONG"
)

type Message struct {
	Type    MessageType
	From    Contact
	To      Contact // Do i need to include the To field in the Ping message?
	Payload []byte
}

// type Ping struct {
// 	Message
// }

func NewPingMessage(from Contact, to Contact) *Message {
	return &Message{
		Type: PING,
		From: from,
		To:   to,
	}
}

func NewPongMessage(from Contact, to Contact) *Message {
	return &Message{
		Type: PONG,
		From: from,
		To:   to,
	}
}
