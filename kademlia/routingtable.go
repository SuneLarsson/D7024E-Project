package kademlia

import (
	"fmt"
	"strings"
)

const bucketSize = 20

// RoutingTable definition
// keeps a refrence contact of me and an array of buckets
type RoutingTable struct {
	me      Contact
	buckets [IDLength * 8]*bucket

	ops chan RoutingRequest
}
type RoutingNode struct {
	prefix string
	bucket *bucket
	child  []*RoutingNode
}

// NewRoutingTable returns a new instance of a RoutingTable
func NewRoutingTable(me Contact) *RoutingTable {
	routingTable := &RoutingTable{
		me:  me,
		ops: make(chan RoutingRequest),
	}
	for i := 0; i < IDLength*8; i++ {
		routingTable.buckets[i] = newBucket()
	}
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
			routingTable.buckets[idx].AddContact(req.contact)
			if req.responseCh != nil {
				req.responseCh <- true
			}

		case FindClosestContacts:
			contacts := routingTable.findClosestContactsInternal(req.target, req.count)
			req.responseCh <- contacts
		}
	}
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

	for i := 1; (bucketIndex-i >= 0 || bucketIndex+i < IDLength*8) && candidates.Len() < count; i++ {
		if bucketIndex-i >= 0 {
			bucket = routingTable.buckets[bucketIndex-i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
		if bucketIndex+i < IDLength*8 {
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
	distance := id.CalcDistance(routingTable.me.ID)
	for i := 0; i < IDLength; i++ {
		for j := 0; j < 8; j++ {
			if (distance[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}

	return IDLength*8 - 1
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

func (node *RoutingNode) PrintTree(indent string, isTail bool) string {
	var result strings.Builder

	branch := "├── "
	if isTail {
		branch = "└── "
	}

	if node.prefix == "" {
		result.WriteString(fmt.Sprintf("[Root]\n"))
	} else {
		result.WriteString(fmt.Sprintf("%s%s%s*\n", indent, branch, node.prefix))
	}
	if node.bucket != nil {
		if node.bucket.Len() == 0 {
			result.WriteString(fmt.Sprintf("%s    <empty>\n", indent))
		} else {
			for e := node.bucket.list.Front(); e != nil; e = e.Next() {
				contact := e.Value.(Contact)
				result.WriteString(fmt.Sprintf("%s    └── %s\n", indent, contact.String()))

			}

		}
		return result.String()
	}
	newIndent := indent
	if isTail {
		newIndent += "    "
	} else {
		newIndent += "│   "
	}
	for i, child := range node.child {
		isLast := (i == len(node.child)-1)
		result.WriteString(child.PrintTree(newIndent, isLast))

	}
	return result.String()

}
