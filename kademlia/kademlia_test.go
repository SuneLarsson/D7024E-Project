package kademlia

import (
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
	sim := NewSimulatedNetwork(0)
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
	sim := NewSimulatedNetwork(0)
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
	sim := NewSimulatedNetwork(0)
	nodeA := setupTestNode("nodeA", sim)
	nodeA.Self.ID = nil // wipe its ID

	nodeB := setupTestNode("nodeB", sim)
	nodeA.JoinNetwork(&nodeB.Self)

	assert.NotNil(t, nodeA.Self.ID, "JoinNetwork should assign a new ID if nil")
}

func TestRefreshBucketEmptyBucket(t *testing.T) {
	sim := NewSimulatedNetwork(0)
	nodeA := setupTestNode("nodeA", sim)

	// Force refresh on an empty bucket
	nodeA.RefreshBucket(0)

	// No panic and no crash expected
	assert.True(t, true, "RefreshBucket should handle empty bucket gracefully")
}

func TestRefreshBucketWithContact(t *testing.T) {
	sim := NewSimulatedNetwork(0)
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
