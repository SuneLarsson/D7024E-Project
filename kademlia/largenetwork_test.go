package kademlia

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func SetupLargeNetwork(t *testing.T, numNodes int, dropRate float64) ([]*Kademlia, *SimulatedNetwork) {

	// 1. Configuration
	sim := NewSimulatedNetwork(dropRate)
	nodes := make([]*Kademlia, numNodes)

	// 2. Node Creation with Deterministic IDs
	for i := 0; i < numNodes; i++ {
		addr := fmt.Sprintf("node-%d", i)
		node := NewTestKademliaNode(addr, sim)

		// Overwrite the random ID with a deterministic, sequential one.
		// This gives us a predictable keyspace to test against.
		idString := fmt.Sprintf("%040x", i)
		node.Self.ID = NewKademliaID(idString)
		nodes[i] = node
	}

	// 4. Bootstrapping
	// To make the network aware of itself, we connect every node to the first node.
	// This simulates a real-world scenario where a new node connects to a known bootstrap node.
	bootstrapNode := nodes[0]
	for i := 1; i < numNodes; i++ {
		nodes[i].RoutingTable.AddContact(bootstrapNode.Self)
		bootstrapNode.RoutingTable.AddContact(nodes[i].Self)
	}

	return nodes, sim
}

func TestLargeNetworkLookupNoDrops(t *testing.T) {
	nodes, _ := SetupLargeNetwork(t, 1000, 0.0)

	// 5. The Test Itself: Find a key that is very "close" to a specific node.
	nodeA := nodes[500] // The node performing the lookup

	targetNode := nodes[11]

	fmt.Println("Target node ID for lookup:", targetNode.Self.ID.String())
	var foundContacts []Contact
	lookupSuccess := assert.Eventually(t, func() bool {
		// Perform the lookup to find the contacts closest to our synthetic targetID.
		contacts := nodeA.IterativeFindNode(targetNode.Self.ID, ALPHA, K)

		if len(contacts) == 0 {
			return false // Keep trying if we haven't found anyone yet
		}
		fmt.Printf("Lookup found %d contacts: %v\n", len(contacts), idsOf(contacts))
		foundContacts = contacts
		return true
	}, 5*time.Second, 100*time.Millisecond, "Lookup should eventually return some contacts")

	assert.True(t, lookupSuccess, "IterativeFindNode failed to complete in time")
	assert.NotEmpty(t, foundContacts, "Lookup should return at least one contact")

	closestContact := foundContacts[0]

	assert.True(t, closestContact.ID.Equals(targetNode.Self.ID), "The closest node found should be the one whose ID we based our target on")
}

func TestLargeNetworkLookupWithDrops(t *testing.T) {
	nodes, _ := SetupLargeNetwork(t, 1000, 0.1)

	// 5. The Test Itself: Find a key that is very "close" to a specific node.
	nodeA := nodes[500] // The node performing the lookup

	targetNode := nodes[11]

	fmt.Println("Target node ID for lookup:", targetNode.Self.ID.String())
	foundContacts := nodeA.IterativeFindNode(targetNode.Self.ID, ALPHA, K)

	// assert.True(t, lookupSuccess, "IterativeFindNode failed to complete in time")
	assert.NotEmpty(t, foundContacts, "Lookup should return at least one contact")

	// Now, verify that the closest node found is the original targetNode.
	// Since IterativeFindNode returns a sorted list, the first contact should be the closest.
	closestContact := foundContacts[0]
	fmt.Printf("Closest contact found: ID=%s, Address=%s\n", closestContact.ID.String(), closestContact.Address)
	fmt.Printf("Target node ID: %s\n", targetNode.Self.ID.String())
	assert.True(t, closestContact.ID.Equals(targetNode.Self.ID), "The closest node found should be the one whose ID we based our target on")
}
