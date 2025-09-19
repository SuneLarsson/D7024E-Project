package kademlia

type rtType int

const (
	AddContact rtType = iota
	FindClosestContacts
)

type RoutingRequest struct {
	requestType rtType
	contact     Contact
	target      *KademliaID
	count       int
	responseCh  chan interface{}
}
