package kademlia

import (
	"reflect"
	"testing"
)

// TestGetBucketIndex verifies that the bucket index is calculated correctly based on the distance.
func TestGetBucketIndex(t *testing.T) {
	// 1. Setup: Create a "me" contact with a zero-ID for easy distance calculation.
	me := NewContact(NewKademliaID("0000000000000000000000000000000000000000"), "localhost:8000")
	rt := NewRoutingTable(me)

	// 2. Test Cases: Define different target IDs and their expected bucket index.
	testCases := []struct {
		name       string
		targetID   *KademliaID
		expected   int
		shouldFail bool
	}{
		{
			name:     "ID with MSB at index 0",
			targetID: NewKademliaID("8000000000000000000000000000000000000000"),
			expected: 0, // Should be in the first bucket
		},
		{
			name:     "ID with MSB at index 7",
			targetID: NewKademliaID("0100000000000000000000000000000000000000"),
			expected: 7,
		},
		{
			name:     "ID with MSB at index 8",
			targetID: NewKademliaID("0080000000000000000000000000000000000000"),
			expected: 8,
		},
		{
			name:     "ID identical to 'me' (distance is 0)",
			targetID: NewKademliaID("0000000000000000000000000000000000000000"),
			expected: 159, // Should fall in the last bucket
		},
		{
			name:     "A distant ID",
			targetID: NewKademliaID("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
			expected: 0, // Distance is all 1s, so MSB is at index 0
		},
		{
			name:       "Incorrect index check",
			targetID:   NewKademliaID("8000000000000000000000000000000000000000"),
			expected:   5, // This is intentionally wrong
			shouldFail: true,
		},
	}

	// 3. Action and Assertion: Run through the test cases.
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bucketIndex := rt.getBucketIndex(tc.targetID)
			if tc.shouldFail {
				if bucketIndex == tc.expected {
					t.Errorf("Expected bucket index to NOT be %d, but it was", tc.expected)
				}
			} else {
				if bucketIndex != tc.expected {
					t.Errorf("Expected bucket index %d, but got %d", tc.expected, bucketIndex)
				}
			}
		})
	}
}

// TestFindClosestContacts tests the core functionality of finding and sorting contacts.
func TestFindClosestContacts(t *testing.T) {
	// 1. Setup: Create a routing table and a set of contacts to add.
	me := NewContact(NewKademliaID("0000000000000000000000000000000000000000"), "localhost:8000")
	rt := NewRoutingTable(me)

	// Create contacts. Their hex values are chosen to control their distance.
	// 8... -> distance 8..., bucket 0
	contact1 := NewContact(NewKademliaID("8000000000000000000000000000000000000001"), "localhost:8001")
	// 4... -> distance 4..., bucket 1
	contact2 := NewContact(NewKademliaID("4000000000000000000000000000000000000002"), "localhost:8002")
	// F... -> distance F..., bucket 0
	contact3 := NewContact(NewKademliaID("F000000000000000000000000000000000000003"), "localhost:8003")
	// 0... -> distance 0..., bucket 7
	contact4 := NewContact(NewKademliaID("0100000000000000000000000000000000000004"), "localhost:8004")
	// 0... -> distance 0..., bucket 15
	contact5 := NewContact(NewKademliaID("0001000000000000000000000000000000000005"), "localhost:8005")

	rt.AddContact(contact1)
	rt.AddContact(contact2)
	rt.AddContact(contact3)
	rt.AddContact(contact4)
	rt.AddContact(contact5)

	// Sub-test 1: Find a limited number of contacts and check for correct sorting.
	t.Run("Finds and sorts the 3 closest contacts", func(t *testing.T) {
		// Target is very close to contact3
		target := NewKademliaID("F000000000000000000000000000000000000000")

		// Expected order by distance from target (F...):
		// 1. contact3 (F... ^ F... = 0)
		// 2. contact1 (F... ^ 8... = 7...)
		// 3. contact2 (F... ^ 4... = B...)
		expectedOrder := []Contact{contact3, contact1, contact2}

		closest := rt.FindClosestContacts(target, 3)

		if len(closest) != 3 {
			t.Fatalf("Expected 3 contacts, but got %d", len(closest))
		}

		// Corrected Assertion: Compare the sorted IDs instead of the structs themselves.
		expectedIDs := getContactIDs(expectedOrder)
		closestIDs := getContactIDs(closest)

		if !reflect.DeepEqual(closestIDs, expectedIDs) {
			t.Errorf("Contacts not sorted correctly by distance.")
			t.Logf("Expected: %v", expectedIDs)
			t.Logf("Got:      %v", closestIDs)
		}
	})

	// Sub-test 2: Request more contacts than are available in the table.
	t.Run("Returns all contacts if count is larger than table size", func(t *testing.T) {
		target := NewKademliaID("0000000000000000000000000000000000000000")
		// We have 5 contacts, ask for 20
		closest := rt.FindClosestContacts(target, 20)

		if len(closest) != 5 {
			t.Fatalf("Expected all 5 contacts, but got %d", len(closest))
		}
	})

	// Sub-test 3: Test on a completely empty routing table.
	t.Run("Returns empty slice for an empty routing table", func(t *testing.T) {
		emptyRt := NewRoutingTable(me)
		closest := emptyRt.FindClosestContacts(me.ID, 5)

		if len(closest) != 0 {
			t.Fatalf("Expected 0 contacts from an empty table, but got %d", len(closest))
		}
	})
}

// Helper function to extract IDs for easier debugging printouts.
func getContactIDs(contacts []Contact) []string {
	ids := make([]string, len(contacts))
	for i, c := range contacts {
		ids[i] = c.ID.String()
	}
	return ids
}
