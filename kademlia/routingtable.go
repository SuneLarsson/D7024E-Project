package kademlia

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// bucketSize defines the maximum number of contacts a single bucket can hold.
const bucketSize = 20

// RoutingTable maintains the local node's view of the network.
// It keeps a reference to the local contact (me), an ordered set of buckets
// that partition the ID space, and an ops channel used to serialize routing
// table mutations and queries.
//
// All modifying operations are funneled through the ops channel so that bucket
// splits and contact additions happen in a single goroutine, avoiding races.
// Lookups are served via the same channel to provide a consistent view.
//
// RoutingTable definition
// keeps a refrence contact of me and an array of buckets
type RoutingTable struct {
	me           Contact
	buckets      []*bucket
	bucketsMutex sync.RWMutex
	ops          chan RoutingRequest
}

// NewRoutingTable returns a new instance of a RoutingTable
func NewRoutingTable(kademlia *Kademlia) *RoutingTable {
	routingTable := &RoutingTable{
		me:      kademlia.Self,
		buckets: make([]*bucket, 0, 160),
		ops:     make(chan RoutingRequest),
	}
	routingTable.bucketsMutex.Lock()
	routingTable.buckets = append(routingTable.buckets, newBucket("", 0))
	routingTable.bucketsMutex.Unlock()
	// routingTable.me = me
	// return routingTable
	time.AfterFunc(200*time.Millisecond, func() { routingTable.run(kademlia) })
	return routingTable
}

// run is the internal event loop that processes routing table operations
// sent over the ops channel. It serializes bucket splits, contact additions,
// and closest-contact queries in a single goroutine.
func (routingTable *RoutingTable) run(kademlia *Kademlia) {
	for req := range routingTable.ops {
		switch req.requestType {
		case AddContact:
			idx := routingTable.getBucketIndex(req.contact.ID)
			canAdd, canSplit := routingTable.buckets[idx].CanAddContact(req.contact, kademlia)

			if !canAdd && canSplit {
				canAdd, idx = routingTable.splitBucket(idx, req.contact, kademlia)
			}

			if canAdd {
				routingTable.buckets[idx].AddContact(req.contact)
			}
			if req.responseCh != nil {
				req.responseCh <- true
			}

		case FindClosestContacts:
			contacts := routingTable.findClosestContactsInternal(req.target, req.count)
			req.responseCh <- contacts
		}
	}
}

// splitBucket splits the bucket at the given index to try to accommodate the
// provided contact. It continues splitting as long as allowed by the split
// policy, then returns whether the contact can be added and the final bucket
// index where it fits (or the last attempted index when not addable).
func (routingTable *RoutingTable) splitBucket(idx int, contact Contact, me *Kademlia) (bool, int) {
	canAdd := false
	canSplit := true
	var bucketsToAdd []*bucket
	indexStart := idx
	routingTable.bucketsMutex.RLock()
	currentBucket := routingTable.buckets[indexStart]
	depth := currentBucket.depth
	for !canAdd && canSplit {
		bucket1, bucket0 := currentBucket.SplitBucket()
		time.Sleep(10 * time.Millisecond)
		bucketsToAdd = append(bucketsToAdd, bucket1, bucket0)

		canAdd, canSplit = bucket1.CanAddContact(contact, me)

		if !canAdd && !canSplit {
			canAdd, canSplit = bucket0.CanAddContact(contact, me)
			currentBucket = bucket0
			idx = indexStart + len(bucketsToAdd) - 1
		} else {
			idx = indexStart + len(bucketsToAdd) - 2
			currentBucket = bucket1
		}

	}
	routingTable.bucketsMutex.RUnlock()
	// If no new buckets are added, then we don't change the current routingtable
	if canAdd {
		routingTable.bucketsMutex.Lock()
		routingTable.buckets[indexStart] = bucketsToAdd[0]
		routingTable.buckets = append(routingTable.buckets[:indexStart+1], append(bucketsToAdd[1:], routingTable.buckets[indexStart+1:]...)...)
		routingTable.bucketsMutex.Unlock()
	} else {
		routingTable.bucketsMutex.Lock()
		routingTable.buckets[indexStart].depth = depth
		routingTable.bucketsMutex.Unlock()
	}
	return canAdd, idx
}

// AddContact adds a new contact to the appropriate bucket by enqueuing a
// request on the ops channel. The actual modification occurs in the run loop.
func (routingTable *RoutingTable) AddContact(contact Contact) {
	// bucketIndex := routingTable.getBucketIndex(contact.ID)
	// bucket := routingTable.buckets[bucketIndex]
	// bucket.AddContact(contact)
	routingTable.ops <- RoutingRequest{
		requestType: AddContact,
		contact:     contact,
	}
}

// FindClosestContacts finds the count closest Contacts to the target in the RoutingTable
func (routingTable *RoutingTable) FindClosestContacts(target *KademliaID, count int) []Contact {

	// return candidates.GetContacts(count)
	respCh := make(chan interface{})
	routingTable.ops <- RoutingRequest{
		requestType: FindClosestContacts,
		target:      target,
		count:       count,
		responseCh:  respCh,
	}
	contacts := (<-respCh).([]Contact)
	return contacts
}

// findClosestContactsInternal scans outward from the bucket containing target
// to neighboring buckets, collecting contacts and sorting by XOR distance. It
// returns up to count closest contacts.
func (routingTable *RoutingTable) findClosestContactsInternal(target *KademliaID, count int) []Contact {
	var candidates ContactCandidates
	bucketIndex := routingTable.getBucketIndex(target)
	bucket := routingTable.buckets[bucketIndex]

	candidates.Append(bucket.GetContactAndCalcDistance(target))

	for i := 1; (bucketIndex-i >= 0 || bucketIndex+i < len(routingTable.buckets)) && candidates.Len() < count; i++ {
		if bucketIndex-i >= 0 {
			bucket = routingTable.buckets[bucketIndex-i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
		if bucketIndex+i < len(routingTable.buckets) {
			bucket = routingTable.buckets[bucketIndex+i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
	}

	candidates.Sort()

	if count > candidates.Len() {
		count = candidates.Len()
	}

	return candidates.GetContacts(count)
}

// getBucketIndex returns the index of the bucket that contains the given ID.
// It iterates over the current bucket ranges to locate the one that includes id.
func (routingTable *RoutingTable) getBucketIndex(id *KademliaID) int {
	/*
		distance := id.CalcDistance(routingTable.me.ID)
		for i := 0; i < IDLength; i++ {
			for j := 0; j < 8; j++ {
				if (distance[i]>>uint8(7-j))&0x1 != 0 {
					return i*8 + j
				}
			}
		}*/
	routingTable.bucketsMutex.RLock()
	defer routingTable.bucketsMutex.RUnlock()
	for i := 0; i < len(routingTable.buckets); i++ {
		bucket := routingTable.buckets[i]
		if bucket.ContainsID(id) {
			return i
		}
	}

	return len(routingTable.buckets) - 1
}

// String returns a human-readable representation of the routing table,
// listing non-empty buckets and their contacts.
func (rt *RoutingTable) String() string {
	result := "Routing Table:\n"
	for i, bucket := range rt.buckets {
		if bucket.Len() > 0 {
			result += fmt.Sprintf("Bucket %d:\n", i)
			bucket.mu.Lock()
			for e := bucket.list.Front(); e != nil; e = e.Next() {
				c := e.Value.(Contact)
				result += fmt.Sprintf("  %s\n", c.String())
			}
			bucket.mu.Unlock()
		}
	}
	return result
}

// PrintTree returns an ASCII representation of the routing table's buckets and
// contained contacts, useful for debugging the bucket split tree.
func (routingTable *RoutingTable) PrintTree() string {
	var result strings.Builder
	branch := "├── "
	tail := "└── "
	basicIndent := "    "

	result.WriteString("[Root]\n")

	for _, bucket := range routingTable.buckets {
		result.WriteString(basicIndent + branch + " " + bucket.prefix + "* depth=" + strconv.Itoa(bucket.depth) + "\n")

		bucket.mu.Lock()
		if bucket.list.Len() == 0 {
			result.WriteString(basicIndent + basicIndent + "<empty>\n")
		} else {
			for e := bucket.list.Front(); e != nil; e = e.Next() {
				nodeID := e.Value.(Contact).ID.String()
				name := e.Value.(Contact).Address
				result.WriteString(basicIndent + basicIndent + tail + "contact(\"" + nodeID + "\", \"" + name + "\")\n")
			}
		}
		bucket.mu.Unlock()

	}

	return result.String()
}
