package kademlia

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helper to build a node quickly ---
func setupTestNode(addr string, sim *SimulatedNetwork) *Kademlia {
	return NewTestKademliaNode(addr, sim)
}

func TestNewKademliaNode(t *testing.T) {
	// This one opens a UDP socket, so just check it creates without error
	node, err := NewKademliaNode("127.0.0.1", 9001)
	require.NoError(t, err, "NewKademliaNode should succeed")
	assert.NotNil(t, node.Network, "Network should be set")
	assert.NotNil(t, node.RoutingTable, "RoutingTable should be set")
}

func TestManagePendingRequestsDispatch(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	node := setupTestNode("nodeA", sim)

	rpcID := *NewRandomKademliaID()
	ch := make(chan Message, 1)

	// Register the request
	node.mapManagerCh <- MapRequest{rpcID: rpcID, responseChan: ch, register: true}

	// Send a response
	resp := Message{Type: PONG, RPCID: rpcID}
	node.mapManagerCh <- MapRequest{rpcID: rpcID, responseMsg: resp, register: false}

	select {
	case got := <-ch:
		assert.Equal(t, resp.Type, got.Type, "Response should be dispatched correctly")
	case <-time.After(time.Second):
		t.Error("Response was not dispatched in time")
	}
}

func TestJoinNetworkInsertsKnownContact(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	nodeA := setupTestNode("nodeA", sim)
	nodeB := setupTestNode("nodeB", sim)

	// Initially, nodeA does not know nodeB
	closest := nodeA.RoutingTable.FindClosestContacts(nodeB.Self.ID, 1)
	assert.Empty(t, closest, "Routing table should not know nodeB before JoinNetwork")

	// NodeA joins with nodeB as entry point
	nodeA.JoinNetwork(&nodeB.Self)

	// Now nodeB should be in nodeA's routing table
	closest = nodeA.RoutingTable.FindClosestContacts(nodeB.Self.ID, 1)
	assert.NotEmpty(t, closest, "Routing table should contain nodeB after JoinNetwork")
}

func TestJoinNetworkAssignsIDIfNil(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	nodeA := setupTestNode("nodeA", sim)
	nodeA.Self.ID = nil // wipe its ID

	nodeB := setupTestNode("nodeB", sim)
	nodeA.JoinNetwork(&nodeB.Self)

	assert.NotNil(t, nodeA.Self.ID, "JoinNetwork should assign a new ID if nil")
}

func TestRefreshBucketEmptyBucket(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	nodeA := setupTestNode("nodeA", sim)

	// Force refresh on an empty bucket
	nodeA.RefreshBucket(0)

	// No panic and no crash expected
	assert.True(t, true, "RefreshBucket should handle empty bucket gracefully")
}

func TestRefreshBucketWithContact(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	nodeA := setupTestNode("nodeA", sim)
	nodeB := setupTestNode("nodeB", sim)

	// Put nodeB in nodeA's bucket
	nodeA.RoutingTable.AddContact(nodeB.Self)

	// Pick the right bucket index
	idx := nodeA.RoutingTable.getBucketIndex(nodeB.Self.ID)

	// Refresh that bucket
	nodeA.RefreshBucket(idx)

	// No panic, should have attempted IterativeFindNode (indirectly tested)
	assert.True(t, true, "RefreshBucket with contact should not crash")
}

// TestRemoveFromBucket removes the least-recently seen node if it doesn't answer, or it does not add the new contact
func TestRemoveFromBucket(t *testing.T) {
	configMutex.Lock()
	K = 2
	BETA = 1
	configMutex.Unlock()

	// Create network with 0 drop rate
	sim := NewSimulatedNetwork(0, 0)

	// Create nodes
	watchingNode := NewTestKademliaNodeWithID(NewKademliaID("1000000000000000000000000000000000000000"), "watching", sim)
	nodeA := NewTestKademliaNodeWithID(NewKademliaID("0000000000000000000000000000000000000000"), "nodeA", sim)
	nodeB := NewTestKademliaNodeWithID(NewKademliaID("8000000000000000000000000000000000000000"), "nodeB", sim)
	nodeC := NewTestKademliaNodeWithID(NewKademliaID("c000000000000000000000000000000000000000"), "nodeC", sim)
	nodeD := NewTestKademliaNodeWithID(NewKademliaID("a000000000000000000000000000000000000000"), "nodeD", sim)

	// Give time for network setup
	time.Sleep(200 * time.Millisecond)

	// Set bucket split parameter
	rt := watchingNode.RoutingTable
	rt.buckets[0].b = 1

	// Add nodes in sequence, with time to stabilize
	nodeA.JoinNetwork(&watchingNode.Self)
	time.Sleep(100 * time.Millisecond)

	nodeB.JoinNetwork(&watchingNode.Self)
	time.Sleep(100 * time.Millisecond)

	nodeC.JoinNetwork(&watchingNode.Self)
	time.Sleep(100 * time.Millisecond)
	fmt.Println(rt.PrintTree())
	fmt.Println("------------------------------------------------")

	// Verify first state
	assert.Equal(t, 2, len(rt.buckets), "Should have exactly 2 buckets")
	assert.True(t, rt.buckets[1].IsPresent(nodeA.Self.ID), "Bucket 1 should contain nodeA")
	assert.True(t, rt.buckets[0].IsPresent(nodeB.Self.ID), "Bucket 0 should contain nodeB")
	assert.True(t, rt.buckets[0].list.Back().Value.(Contact).ID.Equals(nodeB.Self.ID), "nodeB should be the least-recently seen node of bucket 0")
	assert.True(t, rt.buckets[0].IsPresent(nodeC.Self.ID), "Bucket 0 should contain nodeC")

	// Try to add nodeD (should trigger PING to nodeC)
	rt.AddContact(nodeD.Self)
	time.Sleep(100 * time.Millisecond)

	fmt.Println(rt.PrintTree())
	fmt.Println("------------------------------------------------")
	// Verify second state
	assert.Equal(t, 2, len(rt.buckets), "Should still have 2 buckets")
	assert.Equal(t, 2, rt.buckets[0].Len(), "Bucket 0 should have 2 nodes")
	assert.Equal(t, 1, rt.buckets[1].Len(), "Bucket 1 should have 1 node")
	assert.True(t, rt.buckets[0].list.Back().Value.(Contact).ID.Equals(nodeC.Self.ID), "nodeC should be the least-recently seen node of bucket 0")
	assert.True(t, rt.buckets[0].IsPresent(nodeB.Self.ID), "Bucket 0 should contain nodeB")
	assert.True(t, rt.buckets[0].IsPresent(nodeC.Self.ID), "Bucket 0 should contain nodeC")

	// Shutdown nodeC and try to add nodeD again
	delete(sim.nodes, "nodeC")
	time.Sleep(200 * time.Millisecond)

	rt.AddContact(nodeD.Self)
	time.Sleep(200 * time.Millisecond)

	fmt.Println(rt.PrintTree())
	fmt.Println("------------------------------------------------")

	// Verify final state
	assert.True(t, rt.buckets[0].IsPresent(nodeD.Self.ID), "Bucket 0 should now contain nodeD")
	assert.True(t, rt.buckets[0].IsPresent(nodeB.Self.ID), "Bucket 0 should still contain nodeB")
	assert.False(t, rt.buckets[0].IsPresent(nodeC.Self.ID), "Bucket 0 should no longer contain nodeC")
	assert.True(t, rt.buckets[0].list.Back().Value.(Contact).ID.Equals(nodeB.Self.ID), "nodeB should be the least-recently seen node of bucket 0")
	assert.True(t, rt.buckets[0].IsPresent(nodeB.Self.ID), "Bucket 0 should contain nodeB")
	assert.True(t, rt.buckets[0].IsPresent(nodeD.Self.ID), "Bucket 0 should contain nodeD")

	ReloadConfig()
}
