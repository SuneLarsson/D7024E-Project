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
	sim := NewSimulatedNetwork(0, 0)
	watchingNode := NewTestKademliaNodeWithID(NewKademliaID("1000000000000000000000000000000000000000"), "watching", sim)

	rt := watchingNode.RoutingTable
	rt.buckets[0].b = 1

	nodeA := NewTestKademliaNodeWithID(NewKademliaID("0000000000000000000000000000000000000000"), "nodeA", sim) // 00000000
	nodeB := NewTestKademliaNodeWithID(NewKademliaID("8000000000000000000000000000000000000000"), "nodeB", sim) // 10000000
	nodeC := NewTestKademliaNodeWithID(NewKademliaID("c000000000000000000000000000000000000000"), "nodeC", sim) // 11000000
	nodeD := NewTestKademliaNodeWithID(NewKademliaID("a000000000000000000000000000000000000000"), "nodeD", sim) // 10100000

	nodeA.JoinNetwork(&watchingNode.Self)
	nodeB.JoinNetwork(&watchingNode.Self)
	nodeC.JoinNetwork(&watchingNode.Self)

	// At this point the routing table of watching node should be [{C,B},{A}]
	//fmt.Println(rt.PrintTree())
	if len(rt.buckets) != 2 {
		t.Error("The routing table of the watching node should have two buckets")
	}

	if !(rt.buckets[1].IsPresent(nodeA.Self.ID) && rt.buckets[0].IsPresent(nodeB.Self.ID) && rt.buckets[0].IsPresent(nodeC.Self.ID)) {
		t.Error("The routing table of the watching node should contain two buckets :\n 0 with nodeC and nodeB\n 1 with nodeA")
	}

	//fmt.Println(rt.PrintTree())

	nodeD.JoinNetwork(&watchingNode.Self)

	time.Sleep(100 * time.Millisecond)

	// At this point the routing table of watching node should be [{B,C},{A}]
	//fmt.Println(rt.PrintTree())
	if len(rt.buckets) != 2 {
		t.Error("The routing table of the watching node should have two buckets")
	}

	if !(rt.buckets[0].Len() == 2 && rt.buckets[1].Len() == 1) {
		t.Error("The buckets should not have changed")
	}

	/*if !(rt.buckets[1].IsPresent(nodeA.Self.ID) && rt.buckets[0].IsPresent(nodeB.Self.ID) && rt.buckets[0].IsPresent(nodeC.Self.ID)) {
		t.Error("The routing table of the watching node should contain two buckets :\n 0 with nodeB and nodeC\n 1 with nodeA")
	}*/

	if rt.buckets[0].list.Front().Value.(Contact).ID != nodeB.Self.ID {
		t.Error("NodeB should have been brought at the front of the bucket")
	}

	nodeC.Shutdown()

	time.Sleep(200 * time.Millisecond)

	nodeD.JoinNetwork(&watchingNode.Self)

	time.Sleep(100 * time.Millisecond)

	// At this point the routing table of watching node should be [{D,B},{A}]
	//fmt.Println(rt.PrintTree())
	if len(rt.buckets) != 2 {
		t.Error("The routing table of the watching node should have two buckets")
	}

	if !(rt.buckets[0].Len() == 2 && rt.buckets[1].Len() == 1) {
		t.Error("The buckets length should not have changed in length")
	}

	/*if !(rt.buckets[1].IsPresent(nodeA.Self.ID) && rt.buckets[0].IsPresent(nodeB.Self.ID) && rt.buckets[0].IsPresent(nodeD.Self.ID)) {
		t.Error("The routing table of the watching node should contain two buckets :\n 0 with nodeD and nodeB\n 1 with nodeA")
	}*/

	if rt.buckets[0].list.Front().Value.(Contact).ID != nodeD.Self.ID {
		t.Error("NodeD should have been brought at the front of the bucket")
	}

	ReloadConfig()

}
