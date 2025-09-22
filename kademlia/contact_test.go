package kademlia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContactAndString(t *testing.T) {
	id := NewKademliaID("00000000000000000000000000000000000000aa")
	contact := NewContact(id, "127.0.0.1:8000")

	assert.Equal(t, id, contact.ID, "Contact should hold the provided ID")
	assert.Equal(t, "127.0.0.1:8000", contact.Address)
	assert.Contains(t, contact.String(), id.String(), "String() should contain the ID")
	assert.Contains(t, contact.String(), contact.Address, "String() should contain the Address")
}

func TestCalcDistanceAndLess(t *testing.T) {
	id1 := NewKademliaID("0000000000000000000000000000000000000001")
	id2 := NewKademliaID("0000000000000000000000000000000000000002")
	id3 := NewKademliaID("0000000000000000000000000000000000000003")

	c1 := NewContact(id1, "A")
	c2 := NewContact(id2, "B")
	// Calculate distances relative to id3
	c1.CalcDistance(id3)
	c2.CalcDistance(id3)

	assert.NotNil(t, c1.distance, "Distance should be set after CalcDistance")
	assert.NotNil(t, c2.distance, "Distance should be set after CalcDistance")

	// Check Less works (whichever is closer to id3)
	assert.Equal(t, c1.distance.Less(c2.distance), c1.Less(&c2))
}

func TestContactCandidatesAppendAndLen(t *testing.T) {
	candidates := &ContactCandidates{}
	assert.Equal(t, 0, candidates.Len())

	id := NewRandomKademliaID()
	c := NewContact(id, "node")
	candidates.Append([]Contact{c})

	assert.Equal(t, 1, candidates.Len())
	assert.Equal(t, id, candidates.contacts[0].ID)
}

func TestContactCandidatesGetContacts(t *testing.T) {
	candidates := &ContactCandidates{}

	ids := []*KademliaID{
		NewKademliaID("0000000000000000000000000000000000000001"),
		NewKademliaID("0000000000000000000000000000000000000002"),
		NewKademliaID("0000000000000000000000000000000000000003"),
	}

	for i, id := range ids {
		c := NewContact(id, string(rune('A'+i)))
		candidates.Append([]Contact{c})
	}

	got := candidates.GetContacts(2)
	assert.Len(t, got, 2, "Should return requested number of contacts")
	assert.Equal(t, ids[0], got[0].ID)
}

func TestContactCandidatesSwapAndLess(t *testing.T) {
	target := NewKademliaID("000000000000000000000000000000000000000f")

	c1 := NewContact(NewKademliaID("0000000000000000000000000000000000000001"), "A")
	c2 := NewContact(NewKademliaID("0000000000000000000000000000000000000002"), "B")

	// Set distances
	c1.CalcDistance(target)
	c2.CalcDistance(target)

	candidates := &ContactCandidates{}
	candidates.Append([]Contact{c1, c2})

	// Swap them
	candidates.Swap(0, 1)
	assert.Equal(t, c2.ID, candidates.contacts[0].ID)

	// Test Less (should be consistent with contact distances)
	assert.Equal(t, c1.Less(&c2), candidates.Less(1, 0))
}

func TestContactCandidatesSort(t *testing.T) {
	target := NewKademliaID("000000000000000000000000000000000000000f")

	// Make 3 contacts
	c1 := NewContact(NewKademliaID("0000000000000000000000000000000000000001"), "A")
	c2 := NewContact(NewKademliaID("0000000000000000000000000000000000000002"), "B")
	c3 := NewContact(NewKademliaID("0000000000000000000000000000000000000003"), "C")

	// Assign distances relative to target
	c1.CalcDistance(target)
	c2.CalcDistance(target)
	c3.CalcDistance(target)

	candidates := &ContactCandidates{}
	candidates.Append([]Contact{c3, c1, c2})

	candidates.Sort()

	// Check that contacts are sorted by closeness to target
	for i := 0; i < candidates.Len()-1; i++ {
		assert.True(t, !candidates.contacts[i+1].Less(&candidates.contacts[i]),
			"Contacts should be sorted by distance")
	}
}
