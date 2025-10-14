package kademlia

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupHandlerNodes(sim *SimulatedNetwork) (*Kademlia, *Kademlia) {
	nodeA := NewTestKademliaNode("nodeA", sim)
	nodeB := NewTestKademliaNode("nodeB", sim)
	nodeA.RoutingTable.AddContact(nodeB.Self)
	nodeB.RoutingTable.AddContact(nodeA.Self)
	return nodeA, nodeB
}

func TestHandlePingAndPong(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	nodeA, nodeB := setupHandlerNodes(sim)

	err := nodeA.SendPing(&nodeB.Self)
	require.NoError(t, err, "Ping should succeed")

	// Ensure NodeB added NodeA to its routing table
	found := false
	for _, c := range nodeB.RoutingTable.FindClosestContacts(nodeA.Self.ID, 5) {
		if c.ID.Equals(nodeA.Self.ID) {
			found = true
		}
	}
	assert.True(t, found, "Ping should cause receiver to add sender to routing table")
}

func TestHandleStoreSuccess(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	nodeA, nodeB := setupHandlerNodes(sim)

	value := "storeMe"
	hash := sha1Key(value)

	ok := nodeA.Store(&nodeB.Self, value, hash, true)
	assert.True(t, ok, "Store should succeed")

	stored, exists := nodeB.DataStore.Get(hash)
	assert.True(t, exists, "NodeB should store the value")
	assert.Equal(t, value, stored)
}

func TestHandleStoreInvalidPayload(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	_, nodeB := setupHandlerNodes(sim)

	badPayload, _ := json.Marshal(12345)
	msg := &Message{
		Type:    STORE,
		From:    nodeB.Self,
		To:      nodeB.Self,
		Payload: badPayload,
		RPCID:   *NewRandomKademliaID(),
	}

	nodeB.HandleMessage(*msg, nil)

	assert.Equal(t, 0, nodeB.DataStore.Size(), "Invalid payload should not be stored")
}

func TestHandleFindValueFound(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	nodeA, nodeB := setupHandlerNodes(sim)

	value := "value123"
	key := sha1Key(value)
	nodeB.DataStore.Put(key, value, true, true)

	_, found, val := nodeA.FindValue(&nodeB.Self, NewKademliaID(key))
	require.True(t, found, "Value should be found")
	assert.Equal(t, value, *val)
}

func TestHandleFindValueNotFoundReturnsContacts(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	nodeA, nodeB := setupHandlerNodes(sim)

	target := NewRandomKademliaID()
	contacts, found, val := nodeA.FindValue(&nodeB.Self, target)

	assert.False(t, found, "Value not present, should not be found")
	assert.Nil(t, val, "No value should be returned")
	assert.NotEmpty(t, contacts, "Should return contacts instead")
}

func TestHandleFindValueInvalidPayload(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	_, nodeB := setupHandlerNodes(sim)

	badPayload, _ := json.Marshal(12345)
	msg := &Message{
		Type:    FIND_VALUE,
		From:    nodeB.Self,
		To:      nodeB.Self,
		Payload: badPayload,
		RPCID:   *NewRandomKademliaID(),
	}

	nodeB.HandleMessage(*msg, nil)

	assert.Equal(t, 0, nodeB.DataStore.Size(), "Invalid FIND_VALUE payload should not affect datastore")
}

func TestHandleFindNodeReturnsContacts(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	nodeA, nodeB := setupHandlerNodes(sim)

	target := nodeB.Self.ID
	contacts, ok, _ := nodeA.FindNode(&nodeB.Self, target)

	assert.True(t, ok, "FindNode should succeed")
	assert.NotEmpty(t, contacts, "Contacts should be returned")
}

func TestHandleResponseDispatches(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	node := NewTestKademliaNode("nodeX", sim)

	rpcID := *NewRandomKademliaID()
	ch := make(chan Message, 1)

	// Register a waiting request
	node.mapManagerCh <- MapRequest{rpcID: rpcID, responseChan: ch, register: true}

	// Fake response
	msg := Message{
		Type:  PONG,
		From:  node.Self,
		To:    node.Self,
		RPCID: rpcID,
	}
	node.handleResponse(msg)

	select {
	case got := <-ch:
		assert.Equal(t, msg.Type, got.Type, "Response should be dispatched to waiting channel")
	case <-time.After(time.Second):
		t.Error("Response was not dispatched in time")
	}
}

func TestHandleMessageUnknownType(t *testing.T) {
	sim := NewSimulatedNetwork(0, 0)
	_, node := setupHandlerNodes(sim)

	msg := Message{
		Type: "UNKNOWN_TYPE",
		From: node.Self,
		To:   node.Self,
	}
	// Should not panic
	node.HandleMessage(msg, nil)
}

// helper for consistent key hashing
func sha1Key(value string) string {
	h := sha1.Sum([]byte(value))
	return hex.EncodeToString(h[:])
}
