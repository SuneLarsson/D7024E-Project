// in network_test.go
package kademlia

import (
	"d7024e/storage"
	"errors"
	"math/rand"
	"sync"
	"time"
)

// SimulatedNetwork acts as an in-memory message bus for Kademlia nodes.
type SimulatedNetwork struct {
	nodes    map[string]*Kademlia // Map address string to Kademlia instance
	mu       sync.Mutex
	dropRate float64    // Packet drop probability (0.0 to 1.0)
	rand     *rand.Rand // Random source for packet dropping
	seed     int64
}

// NewSimulatedNetwork creates a new network simulation with a configurable packet drop rate.
// The dropRate should be a value between 0.0 (no drops) and 1.0 (all drops).
func NewSimulatedNetwork(dropRate float64, seed int64) *SimulatedNetwork {
	return &SimulatedNetwork{
		nodes:    make(map[string]*Kademlia),
		dropRate: dropRate,
		rand:     rand.New(rand.NewSource(seed)),
		seed:     seed,
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
	defer m.sim.mu.Unlock()

	// Simulate packet drop
	if m.sim.rand.Float64() < m.sim.dropRate {

		// Packet is "dropped". We return nil to simulate the "fire and forget"
		// nature of UDP, where the sender doesn't know about the drop.
		return nil
	}

	targetNode, found := m.sim.nodes[addr]

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
	return NewTestKademliaNodeWithID(NewRandomKademliaID(), address, sim)
}

func NewTestKademliaNodeWithID(id *KademliaID, address string, sim *SimulatedNetwork) *Kademlia {
	contact := Contact{
		ID:      id,
		Address: address,
	}

	ttl := time.Duration(TTL) * time.Second

	// 1. Create the Kademlia struct instance first.
	kademliaNode := &Kademlia{
		Self:         contact,
		DataStore:    *storage.NewStorage(ttl),
		mapManagerCh: make(chan MapRequest),
		keyStore:     make(map[string]chan string),
		ttl:          ttl,
		alpha:        ALPHA,
		beta:         BETA,
		k:            K,
		done:         make(chan struct{}),
	}

	rt := NewRoutingTable(kademliaNode)

	kademliaNode.RoutingTable = rt

	// 2. Create the mock network adapter for this specific node.
	adapter := &MockNetworkAdapter{
		node: kademliaNode,
		sim:  sim,
	}

	// 3. Assign the adapter to the node's Network field.
	kademliaNode.Network = adapter

	kademliaNode.wg.Add(3)

	// 4. Register the fully assembled node with the central simulation.
	sim.AddNode(kademliaNode)

	go kademliaNode.managePendingRequests()
	go kademliaNode.RunPeriodicCleanup(5 * time.Second)
	go kademliaNode.PeriodicReplication(time.Duration(tReplicate) * time.Hour)
	return kademliaNode
}
