package kademlia

import (
	"fmt"
	"strings"
	"sync"
)

const bucketSize = 20

// RoutingTable definition
// keeps a refrence contact of me and an array of buckets
type RoutingTable struct {
	me           Contact
	buckets      []*bucket
	bucketsMutex sync.RWMutex
	ops          chan RoutingRequest
}

// NewRoutingTable returns a new instance of a RoutingTable
func NewRoutingTable(me Contact) *RoutingTable {
	routingTable := &RoutingTable{
		me:      me,
		buckets: make([]*bucket, 0, 160),
		ops:     make(chan RoutingRequest),
	}
	routingTable.bucketsMutex.Lock()
	routingTable.buckets = append(routingTable.buckets, newBucket("", 0))
	routingTable.bucketsMutex.Unlock()
	// routingTable.me = me
	// return routingTable
	go routingTable.run()
	return routingTable
}

func (routingTable *RoutingTable) run() {
	for req := range routingTable.ops {
		switch req.requestType {
		case AddContact:
			idx := routingTable.getBucketIndex(req.contact.ID)
			canAdd, canSplit := routingTable.buckets[idx].CanAddContact(req.contact, routingTable.me)

			if !canAdd && canSplit {
				canAdd, idx = routingTable.splitBucket(idx, req.contact)
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

// splitBucket splits the bucket at the index to try to accomodate for the contact
func (routingTable *RoutingTable) splitBucket(idx int, contact Contact) (bool, int) {
	var canAdd bool = false
	var canSplit bool = true
	var bucketsToAdd []*bucket
	indexStart := idx
	routingTable.bucketsMutex.RLock()
	currentBucket := routingTable.buckets[indexStart]
	for !canAdd && canSplit {
		bucket1, bucket0 := currentBucket.SplitBucket()
		bucketsToAdd = append(bucketsToAdd, bucket1, bucket0)

		canAdd, canSplit = bucket1.CanAddContact(contact, routingTable.me)

		if !canAdd {
			canAdd, canSplit = bucket0.CanAddContact(contact, routingTable.me)
			idx = indexStart + len(bucketsToAdd) - 1
		} else {
			idx = indexStart + len(bucketsToAdd) - 2
		}

	}
	routingTable.bucketsMutex.RUnlock()
	// If no new buckets are added, then we don't change the current routingtable
	if canAdd {
		routingTable.bucketsMutex.Lock()
		routingTable.buckets[indexStart] = bucketsToAdd[0]
		routingTable.buckets = append(routingTable.buckets[:indexStart+1], append(bucketsToAdd[1:], routingTable.buckets[indexStart+1:]...)...)
		routingTable.bucketsMutex.Unlock()
	}
	return canAdd, idx
}

// AddContact add a new contact to the correct Bucket
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

// getBucketIndex get the correct Bucket index for the KademliaID
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
	for i := 0; i < len(routingTable.buckets); i++ {
		bucket := routingTable.buckets[i]
		if bucket.ContainsID(id) {
			return i
		}
	}

	return len(routingTable.buckets) - 1
}

func (rt *RoutingTable) String() string {
	result := "Routing Table:\n"
	for i, bucket := range rt.buckets {
		if bucket.Len() > 0 {
			result += fmt.Sprintf("Bucket %d:\n", i)
			for e := bucket.list.Front(); e != nil; e = e.Next() {
				c := e.Value.(Contact)
				result += fmt.Sprintf("  %s\n", c.String())
			}
		}
	}
	return result
}

func (routingTable *RoutingTable) PrintTree() string {
	var result strings.Builder
	branch := "├── "
	tail := "└── "
	basicIndent := "    "

	result.WriteString("[Root]\n")

	for _, bucket := range routingTable.buckets {
		result.WriteString(basicIndent + branch + " " + bucket.prefix + "*\n")

		if bucket.list.Len() == 0 {
			result.WriteString(basicIndent + basicIndent + "<empty>\n")
		} else {
			for e := bucket.list.Front(); e != nil; e = e.Next() {
				nodeID := e.Value.(Contact).ID.String()
				name := e.Value.(Contact).Address
				result.WriteString(basicIndent + basicIndent + tail + "contact(\"" + nodeID + "\", \"" + name + "\")\n")
			}
		}

	}

	return result.String()
}
