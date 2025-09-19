package kademlia

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Extract IDs for assertions
func getIDs(contacts []Contact) []string {
	ids := make([]string, len(contacts))
	for i, c := range contacts {
		ids[i] = c.ID.String()
	}
	return ids
}

// Set up two nodes that know each other
func setupTwoNodes(sim *SimulatedNetwork, addrA, addrB string) (*Kademlia, *Kademlia) {
	nodeA := NewTestKademliaNode(addrA, sim)
	nodeB := NewTestKademliaNode(addrB, sim)
	nodeA.RoutingTable.AddContact(nodeB.Self)
	nodeB.RoutingTable.AddContact(nodeA.Self)
	return nodeA, nodeB
}

// Compute the SHA1-based key used by IterativeStore/IterativeFindValue
func hashKeyForValue(value string) *KademliaID {
	h := sha1.Sum([]byte(value))
	return NewKademliaID(hex.EncodeToString(h[:]))
}

func TestIterativeFindNode(t *testing.T) {
	t.Run("Neighbors", func(t *testing.T) {
		sim := NewSimulatedNetwork()
		nodeA, nodeB := setupTwoNodes(sim, "nodeA", "nodeB")
		nodeC := NewTestKademliaNode("nodeC", sim)
		nodeA.RoutingTable.AddContact(nodeC.Self)

		target := NewRandomKademliaID()
		result := nodeA.IterativeFindNode(target, 3, 20)

		ids := getIDs(result)
		assert.Contains(t, ids, nodeB.Self.ID.String(), "Result should contain nodeB")
		assert.Contains(t, ids, nodeC.Self.ID.String(), "Result should contain nodeC")
	})

	t.Run("Empty routing table", func(t *testing.T) {
		sim := NewSimulatedNetwork()
		node := NewTestKademliaNode("nodeX", sim)

		target := NewRandomKademliaID()
		result := node.IterativeFindNode(target, 3, 20)

		assert.Empty(t, result, "Empty routing table should yield no contacts")
	})

	t.Run("Multi-hop discovery", func(t *testing.T) {
		sim := NewSimulatedNetwork()

		nodeA := NewTestKademliaNode("nodeA", sim)
		nodeB := NewTestKademliaNode("nodeB", sim)
		nodeC := NewTestKademliaNode("nodeC", sim)
		nodeD := NewTestKademliaNode("nodeD", sim)

		// A → B → C → D
		nodeA.RoutingTable.AddContact(nodeB.Self)
		nodeB.RoutingTable.AddContact(nodeC.Self)
		nodeC.RoutingTable.AddContact(nodeD.Self)

		result := nodeA.IterativeFindNode(nodeD.Self.ID, 3, 20)
		ids := getIDs(result)

		assert.Contains(t, ids, nodeD.Self.ID.String(), "A should discover D via iterative lookup")
	})

	t.Run("Results sorted by distance", func(t *testing.T) {
		sim := NewSimulatedNetwork()
		target := NewRandomKademliaID()

		nodeA, _ := setupTwoNodes(sim, "nodeA", "nodeB")
		nodeC := NewTestKademliaNode("nodeC", sim)
		nodeA.RoutingTable.AddContact(nodeC.Self)

		result := nodeA.IterativeFindNode(target, 3, 20)

		if len(result) > 1 {
			assert.True(t, result[0].distance.Less(result[1].distance),
				"First contact should be closer to target than second")
		}
	})

}

func TestIterativeFindValue(t *testing.T) {
	t.Run("Found value immediately", func(t *testing.T) {
		sim := NewSimulatedNetwork()
		nodeA, nodeB := setupTwoNodes(sim, "nodeA", "nodeB")

		value := "testValue"
		key := hashKeyForValue(value)
		nodeB.DataStore.Put(key.String(), value)

		_, found := nodeA.IterativeFindValue(key, 3, 20)

		require.NotNil(t, found, "Value should not be nil")
		assert.Equal(t, value, *found)
	})

	t.Run("Not found returns contacts", func(t *testing.T) {
		sim := NewSimulatedNetwork()
		nodeA, _ := setupTwoNodes(sim, "nodeA", "nodeB")

		target := NewRandomKademliaID()
		contacts, found := nodeA.IterativeFindValue(target, 3, 20)

		assert.Nil(t, found)
		assert.NotEmpty(t, contacts)
	})

	t.Run("Caches value in closest node without value", func(t *testing.T) {
		sim := NewSimulatedNetwork()
		nodeA := NewTestKademliaNode("nodeA", sim)
		nodeB := NewTestKademliaNode("nodeB", sim)
		nodeC := NewTestKademliaNode("nodeC", sim)

		value := "closestGetsValue"
		key := hashKeyForValue(value)
		nodeC.DataStore.Put(key.String(), value)

		// Chain: A → B → C
		nodeA.RoutingTable.AddContact(nodeB.Self)
		nodeB.RoutingTable.AddContact(nodeC.Self)

		nodeA.IterativeFindValue(key, 1, 20)

		require.Eventually(t, func() bool {
			stored, _ := nodeB.DataStore.Get(key.String())
			return stored == value
		}, time.Second, 50*time.Millisecond,
			"NodeB should eventually cache the value")
	})

}

func TestIterativeStore(t *testing.T) {
	t.Run("Stores on one neighbor", func(t *testing.T) {
		sim := NewSimulatedNetwork()
		nodeA, nodeB := setupTwoNodes(sim, "nodeA", "nodeB")

		value := "storeMe"
		key, success := nodeA.IterativeStore(value)

		assert.True(t, success, "Store should succeed")
		stored, _ := nodeB.DataStore.Get(key)
		assert.Equal(t, value, stored)
	})

	t.Run("No nodes available", func(t *testing.T) {
		sim := NewSimulatedNetwork()
		nodeA := NewTestKademliaNode("nodeA", sim)

		_, success := nodeA.IterativeStore("nothingHappens")
		assert.False(t, success, "Store should fail when no nodes are available")
	})

	t.Run("Stores on multiple nodes", func(t *testing.T) {
		sim := NewSimulatedNetwork()
		nodeA := NewTestKademliaNode("nodeA", sim)
		nodeB := NewTestKademliaNode("nodeB", sim)
		nodeC := NewTestKademliaNode("nodeC", sim)

		nodeA.RoutingTable.AddContact(nodeB.Self)
		nodeA.RoutingTable.AddContact(nodeC.Self)

		value := "spreadThis"
		key, success := nodeA.IterativeStore(value)

		assert.True(t, success, "Store should succeed with multiple nodes")

		storedB, _ := nodeB.DataStore.Get(key)
		storedC, _ := nodeC.DataStore.Get(key)

		assert.Equal(t, value, storedB)
		assert.Equal(t, value, storedC)
	})
}

func TestHelpers(t *testing.T) {
	t.Run("pickAlpha stops at alpha limit", func(t *testing.T) {
		candidates := &ContactCandidates{}
		target := NewRandomKademliaID()
		alpha := 3
		// Add more contacts than alpha
		for i := 0; i < alpha+5; i++ {
			c := NewContact(NewRandomKademliaID(), "addr")
			c.CalcDistance(target)
			candidates.Append([]Contact{c})
		}

		queried := make(map[string]bool)
		picked := candidates.pickAlpha(queried, alpha)

		assert.Equal(t, alpha, len(picked), "Should only pick up to alpha contacts")
	})
	t.Run("SortWithTarget set  nil distances and sorts", func(t *testing.T) {
		candidates := &ContactCandidates{}
		target := NewRandomKademliaID()

		// Add contacts with nil distances
		for i := 0; i < 5; i++ {
			c := NewContact(NewRandomKademliaID(), "addr")
			candidates.Append([]Contact{c})
		}

		candidates.SortWithTarget(target)
		// Verify all distances are set
		for _, c := range candidates.contacts {
			assert.NotNil(t, c.distance, "Contact distance should be set after SortWithTarget")
		}

		// Verify sorting order
		for i := 0; i < candidates.Len()-1; i++ {
			assert.True(t, !candidates.contacts[i+1].Less(&candidates.contacts[i]),
				"Contacts should be sorted by distance to target")
		}
	})
}
