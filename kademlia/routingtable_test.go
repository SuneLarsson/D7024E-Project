package kademlia

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// TestBucketWorking verifies that the bucket works correctly with its parameters
func TestBucketWorking(t *testing.T) {
	// 1. Setup: Create a "me" contact
	me, _ := NewKademliaNode("localhost", 8000)
	me.Self = NewContact(NewKademliaID("0000000000000000000000000000000000000000"), "localhost:8000")
	K = 3
	BETA = 3
	rt := NewRoutingTable(me)
	defer ReloadConfig()
	rt.buckets[0].b = 3

	rt.AddContact(NewContact(NewKademliaID("9000000000000000000000000000000000000000"), "10010"))
	rt.AddContact(NewContact(NewKademliaID("8800000000000000000000000000000000000000"), "10001"))
	rt.AddContact(NewContact(NewKademliaID("0800000000000000000000000000000000000000"), "00001"))

	rt.bucketsMutex.RLock()
	if len(rt.buckets) > 1 {
		t.Error("There should be only one bucket at this time")
	}
	rt.bucketsMutex.RUnlock()

	rt.AddContact(NewContact(NewKademliaID("8000000000000000000000000000000000000000"), "10000"))

	time.Sleep(100 * time.Millisecond)

	rt.bucketsMutex.RLock()
	if len(rt.buckets) != 2 {
		t.Error("There should be two buckets at this time")
	}

	if rt.buckets[0].Len() != 3 && rt.buckets[0].Len() != 1 {
		t.Error("The bucket at index 0 should contain 3 contacts, while the bucket at index 1 should contain 1")
	}
	rt.bucketsMutex.RUnlock()

	rt.AddContact(NewContact(NewKademliaID("C000000000000000000000000000000000000000"), "11000"))

	time.Sleep(100 * time.Millisecond)

	rt.bucketsMutex.RLock()
	if len(rt.buckets) != 3 {
		t.Error("There should be three buckets at this time")
	}

	bucketSizes := [3]int{1, 3, 1}
	for i, size := range bucketSizes {
		if rt.buckets[i].Len() != size {
			t.Error("Bucket at index", i, "is of size", rt.buckets[i].Len(), "when it should be", size)
		}
	}
	rt.bucketsMutex.RUnlock()

	rt.AddContact(NewContact(NewKademliaID("E000000000000000000000000000000000000000"), "11100"))

	time.Sleep(100 * time.Millisecond)

	rt.bucketsMutex.RLock()
	if len(rt.buckets) != 3 {
		t.Error("There should be three buckets at this time")
	}

	bucketSizes = [3]int{2, 3, 1}
	for i, size := range bucketSizes {
		if rt.buckets[i].Len() != size {
			t.Error("Bucket at index", i, "is of size", rt.buckets[i].Len(), "when it should be", size)
		}
	}
	rt.bucketsMutex.RUnlock()
}

// TestBucketIsPresent verifies that the IsPresent function of a bucket is correct
func TestBucketIsPresent(t *testing.T) {
	me, _ := NewKademliaNode("localhost", 8000)
	me.Self = NewContact(NewKademliaID("0000000000000000000000000000000000000000"), "localhost:8000")
	rt := NewRoutingTable(me)
	rt.AddContact(NewContact(NewKademliaID("c000000000000000000000000000000000000000"), "Present"))

	time.Sleep(100 * time.Millisecond)
	fmt.Println(rt.PrintTree())
	if !rt.buckets[0].IsPresent(NewKademliaID("c000000000000000000000000000000000000000")) {
		t.Error("The contact should be present")
	}

	if rt.buckets[0].IsPresent(NewKademliaID("c100000000000000000000000000000000000000")) {
		t.Error("There is no contact with this KademliaID")
	}
}

// TestGetBucketIndex verifies that the bucket index is correctly gotten
func TestGetBucketIndex(t *testing.T) {
	me, _ := NewKademliaNode("localhost", 8000)
	me.Self = NewContact(NewKademliaID("0000000000000000000000000000000000000000"), "localhost:8000")
	rt := NewRoutingTable(me)
	rt.buckets = []*bucket{
		newBucket("11", 2),
		newBucket("10", 2),
		newBucket("01", 2),
		newBucket("001", 3),
		newBucket("0001", 4),
		newBucket("0000", 4),
	}

	expectedIndexes := []int{0, 5, 2, 4, 3, 1}
	idsToVerify := []*KademliaID{
		NewKademliaID("C000000000000000000000000000000000000000"), //11000...
		NewKademliaID("0800000000000000000000000000000000000000"), //00001...
		NewKademliaID("4800000000000000000000000000000000000000"), //01001...
		NewKademliaID("1800000000000000000000000000000000000000"), //00011...
		NewKademliaID("2800000000000000000000000000000000000000"), //00101...
		NewKademliaID("9800000000000000000000000000000000000000"), //10011...
	}

	for i := 0; i < len(idsToVerify); i++ {
		if res := rt.getBucketIndex(idsToVerify[i]); res != expectedIndexes[i] {
			t.Error("Index of ID", idsToVerify[i].String(), "found at bucket", res, "when expected at", expectedIndexes[i])
		}
	}
}

// TestFindClosestContacts tests the core functionality of finding and sorting contacts.
func TestFindClosestContacts(t *testing.T) {
	// 1. Setup: Create a routing table and a set of contacts to add.
	me, _ := NewKademliaNode("localhost", 8000)
	me.Self = NewContact(NewKademliaID("0000000000000000000000000000000000000000"), "localhost:8000")
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
		closest := emptyRt.FindClosestContacts(me.Self.ID, 5)

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

func TestRoutingTablePrint(t *testing.T) {
	self, _ := NewKademliaNode("localhost", 8000)
	self.Self = NewContact(NewRandomKademliaID(), "nodeA")
	rt := NewRoutingTable(self)

	rt.AddContact(NewContact(NewRandomKademliaID(), "nodeB"))
	rt.AddContact(NewContact(NewRandomKademliaID(), "nodeC"))
	rt.AddContact(NewContact(NewRandomKademliaID(), "nodeD"))

	fmt.Println(rt.String())
}

func TestGeneralRoutingTreePrint(t *testing.T) {
	// Build a fake routing tree for b=2

	self, _ := NewKademliaNode("localhost", 8000)
	self.Self = NewContact(NewRandomKademliaID(), "Myself")
	rt := self.RoutingTable

	rt.bucketsMutex.Lock()
	defer rt.bucketsMutex.Unlock()

	rt.buckets = []*bucket{
		newBucket("00", 2),
		newBucket("01", 2),
		newBucket("10", 2),
		newBucket("11", 2),
	}

	nodeA := NewContact(NewRandomKademliaID(), "nodeA")
	fmt.Println(nodeA.ID.String())
	rt.AddContact(nodeA)
	rt.AddContact(NewContact(NewRandomKademliaID(), "nodeB"))
	rt.AddContact(NewContact(NewRandomKademliaID(), "nodeC"))
	rt.AddContact(NewContact(NewRandomKademliaID(), "nodeD"))

	time.Sleep(100 * time.Millisecond)

	fmt.Println("Routing Tree (b=2):")
	fmt.Println(rt.PrintTree())
	//t.Error("test")
}
