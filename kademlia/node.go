package kademlia

type Node struct {
	ID      *KademliaID
	Address string
	Network *Network
}

func (n *Node) Ping(contact *Contact) {
	n.Network.SendPingMessage(contact)
}

// new Node j join procedure
// must initially know the ID and IP address of one another node c
// Perform a lookup of its own ID
// Perform bucket refresh
func (j *Node) Join(c *Contact) {

}

// Not complete
func NewNode(ip string, port int) *Node {
	network, err := Listen(ip, port)
	if err != nil {
		return nil
	}
	return &Node{
		ID:      network.Self.ID,
		Address: network.Self.Address,
		Network: network,
	}
}
