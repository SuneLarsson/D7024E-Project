package kademlia

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPrimitiveNodes(sim *SimulatedNetwork, addrA, addrB string) (*Kademlia, *Kademlia) {
	nodeA := NewTestKademliaNode(addrA, sim)
	nodeB := NewTestKademliaNode(addrB, sim)
	nodeA.RoutingTable.AddContact(nodeB.Self)
	nodeB.RoutingTable.AddContact(nodeA.Self)
	return nodeA, nodeB
}

func TestFindNodeTimeout(t *testing.T) {
	sim := NewSimulatedNetwork()
	nodeA := NewTestKademliaNode("nodeA", sim)
	badContact := Contact{
		ID:      NewRandomKademliaID(),
		Address: "nonExistentNode",
	}
	target := NewRandomKademliaID()
	contacts, ok, _ := nodeA.FindNode(&badContact, target)
	assert.False(t, ok, "FindNode to nonexistent node should fail")
	assert.Empty(t, contacts, "Contacts should be empty on failure")
}

func TestFindValueTimeout(t *testing.T) {
	sim := NewSimulatedNetwork()
	nodeA := NewTestKademliaNode("nodeA", sim)
	badContact := Contact{
		ID:      NewRandomKademliaID(),
		Address: "nonExistentNode",
	}
	target := NewRandomKademliaID()
	contacts, found, value := nodeA.FindValue(&badContact, target)

	assert.False(t, found, "FindValue to nonexistent node should fail")
	assert.Empty(t, contacts, "Contacts should be empty on failure")
	assert.Nil(t, value, "Value should be nil on failure")
}

func TestSendPing(t *testing.T) {
	sim := NewSimulatedNetwork()
	nodeA, nodeB := setupPrimitiveNodes(sim, "nodeA", "nodeB")

	err := nodeA.SendPing(&nodeB.Self)
	assert.NoError(t, err, "Ping should succeed between connected nodes")
}

func TestFindNodePrimitive(t *testing.T) {
	sim := NewSimulatedNetwork()
	nodeA, nodeB := setupPrimitiveNodes(sim, "nodeA", "nodeB")

	target := nodeB.Self.ID
	contacts, ok, _ := nodeA.FindNode(&nodeB.Self, target)

	require.True(t, ok, "FindNode should return response")
	assert.NotEmpty(t, contacts, "FindNode should return at least one contact")
	foundA := false
	for _, c := range contacts {
		if c.ID.Equals(nodeA.Self.ID) {
			foundA = true
		}
	}
	assert.True(t, foundA, "FindNode response should include nodeA (since nodeB only knows nodeA)")
}

func TestStorePrimitive(t *testing.T) {
	sim := NewSimulatedNetwork()
	nodeA, nodeB := setupPrimitiveNodes(sim, "nodeA", "nodeB")

	value := "helloWorld"
	key := hashKeyForValue(value)
	// hash := sha1.Sum([]byte(value))
	// key := hex.EncodeToString(hash[:])
	ok := nodeA.Store(&nodeB.Self, value, key.String())

	assert.True(t, ok, "Store RPC should succeed")

	// Verify nodeB actually stored the value
	stored, exists := nodeB.DataStore.Get(key.String())
	assert.True(t, exists, "Value should be stored on nodeB")
	assert.Equal(t, value, stored)
}

func TestFindValuePrimitive(t *testing.T) {
	sim := NewSimulatedNetwork()
	nodeA, nodeB := setupPrimitiveNodes(sim, "nodeA", "nodeB")

	value := "secret"
	key := hashKeyForValue(value)
	nodeB.DataStore.Put(key.String(), value)

	contacts, found, gotValue := nodeA.FindValue(&nodeB.Self, key)

	require.True(t, found, "FindValue should succeed in finding value")
	assert.Nil(t, contacts, "Should not return contacts when value found")
	assert.Equal(t, value, *gotValue)
}

func TestFindValueReturnsContactsWhenNotFound(t *testing.T) {
	sim := NewSimulatedNetwork()
	nodeA, nodeB := setupPrimitiveNodes(sim, "nodeA", "nodeB")

	key := NewRandomKademliaID()
	contacts, found, gotValue := nodeA.FindValue(&nodeB.Self, key)

	assert.False(t, found, "Should not find value")
	assert.Nil(t, gotValue, "Value should be nil when not found")
	assert.NotEmpty(t, contacts, "Should return contacts when value not found")
}

func TestPingTimeout(t *testing.T) {
	// NodeA tries to ping a contact not in the network
	sim := NewSimulatedNetwork()
	nodeA := NewTestKademliaNode("nodeA", sim)
	badContact := Contact{
		ID:      NewRandomKademliaID(),
		Address: "nonExistentNode",
	}

	err := nodeA.SendPing(&badContact)
	assert.Error(t, err, "Ping to nonexistent node should timeout or error")
}

func TestStoreTimeout(t *testing.T) {
	// NodeA tries to store on nonexistent node
	sim := NewSimulatedNetwork()
	nodeA := NewTestKademliaNode("nodeA", sim)
	badContact := Contact{
		ID:      NewRandomKademliaID(),
		Address: "nonExistentNode",
	}

	ok := nodeA.Store(&badContact, "val", "key")
	assert.False(t, ok, "Store should fail on nonexistent node")
}

func TestForgetPrimitive(t *testing.T) {
	sim := NewSimulatedNetwork()
	nodeA, _ := setupPrimitiveNodes(sim, "nodeA", "nodeB")
	nodeA.keyStore = make(map[string]chan string)

	value := "helloWorld"
	key, _ := nodeA.IterativeStore(value)

	time.Sleep(100 * time.Millisecond)

	nodeA.keyMutex.Lock()
	assert.True(t, nodeA.keyStore[key] != nil, "Node A should contain the channel to forget")
	nodeA.keyMutex.Unlock()

	nodeA.Forget(key)

	nodeA.keyMutex.Lock()
	assert.True(t, nodeA.keyStore[key] == nil, "Node A should not contain the channel to forget anymore")
	nodeA.keyMutex.Unlock()

}
