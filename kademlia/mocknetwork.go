// in network_test.go
package kademlia

import (
	"errors"
	"sync"
)

// SimulatedNetwork acts as an in-memory message bus for Kademlia nodes.
type SimulatedNetwork struct {
	nodes map[string]*Kademlia // Map address string to Kademlia instance
	mu    sync.Mutex
}

func NewSimulatedNetwork() *SimulatedNetwork {
	return &SimulatedNetwork{
		nodes: make(map[string]*Kademlia),
	}
}

// AddNode registers a Kademlia node with the simulated network.
func (s *SimulatedNetwork) AddNode(node *Kademlia) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node.Self.Address] = node
}

// MockNetworkAdapter is a per-node view of the network that implements NetworkAPI
type MockNetworkAdapter struct {
	node *Kademlia
	sim  *SimulatedNetwork
}

// SendMessage finds the target node in the simulation and calls its handler directly.
func (m *MockNetworkAdapter) SendMessage(addr string, msg *Message) error {
	m.sim.mu.Lock()
	targetNode, found := m.sim.nodes[addr]
	m.sim.mu.Unlock()

	if !found {
		return errors.New("node not found in simulation: " + addr)
	}

	// "Deliver" the message by directly calling the target's handler
	// Run in a goroutine to better simulate real network asynchronicity
	go targetNode.HandleMessage(*msg, nil) // addr is nil, not needed for sim
	return nil
}

// Listen is a no-op in the simulation, as messages are delivered instantly.
func (m *MockNetworkAdapter) Listen() error {
	return nil
}

func NewTestKademliaNode(address string, sim *SimulatedNetwork) *Kademlia {
	contact := Contact{
		ID:      NewRandomKademliaID(),
		Address: address,
	}
	rt := NewRoutingTable(contact)

	// 1. Create the Kademlia struct instance first.
	kademliaNode := &Kademlia{
		Self:         contact,
		RoutingTable: rt,
		DataStore:    make(map[KademliaID]DataItem),
		mapManagerCh: make(chan MapRequest),
	}

	// 2. Create the mock network adapter for this specific node.
	adapter := &MockNetworkAdapter{
		node: kademliaNode,
		sim:  sim,
	}

	// 3. Assign the adapter to the node's Network field.
	kademliaNode.Network = adapter

	// 4. Register the fully assembled node with the central simulation.
	sim.AddNode(kademliaNode)

	go kademliaNode.managePendingRequests()
	return kademliaNode
}
