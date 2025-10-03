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
	buckets []*bucket
	pingFn  func(Contact) bool // Function to ping a contact; returns true if alive

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
	root := newBucket(NewKademliaID("0...0"), 0)
	// routingTable.me = me
	// return routingTable
	routingTable.buckets = []*bucket{root}
	go routingTable.run()
	return routingTable
}

func (routingTable *RoutingTable) run() {
	for req := range routingTable.ops {
		switch req.requestType {
		case AddContact:
			contact := req.contact
			bucketIndex := routingTable.findLeafBucketIndex(contact.ID)
			bucket := routingTable.buckets[bucketIndex]

			bucket.mu.Lock()

			moved := false

			for e := bucket.list.Front(); e != nil; e = e.Next() {
				if e.Value.(Contact).ID.Equals(contact.ID) {
					bucket.list.MoveToFront(e)
					moved = true
					break
				}
			}

			if !moved {
				if bucket.list.Len() < bucketSize {
					bucket.list.PushFront(contact)
				} else {
					// Split or evict
					if bucket.containsID(routingTable.me.ID) {
						bucket.mu.Unlock()
						routingTable.splitBucket(bucketIndex)
						// Retry adding the contact after splitting
						routingTable.ops <- req
						continue
					} else {
						// Evict the least recently seen contact
						// and add the new contact to the front
						// Note: In a full implementation, you would ping the
						// oldest contact to check if it's still alive before eviction
						oldest := bucket.list.Back().Value.(Contact)
						bucket.mu.Unlock()
						alive := routingTable.ping(oldest)
						bucket.mu.Lock()
						if !alive {
							bucket.list.Remove(bucket.list.Back())
							bucket.list.PushFront(contact)
						}

					}
				}
			}
			bucket.mu.Unlock()
			if req.responseCh != nil {
				req.responseCh <- true
			}

		case FindClosestContacts:
			contacts := routingTable.findClosestContactsInternal(req.target, req.count)
			req.responseCh <- contacts
		}
	}
}

func (routingTable *RoutingTable) splitBucket(idx int) {
	oldBucket := routingTable.buckets[idx]

	branchFactor := 1 << BETA

	//Create child buckets
	children := make([]*bucket, branchFactor)
	for i := 0; i < branchFactor; i++ {
		childPrefix := oldBucket.prefix.CloneWithExtraBits(oldBucket.depth, i, BETA)
		children[i] = newBucket(childPrefix, oldBucket.depth+branchFactor)
	}

	// Redistribute the child buckets in place of the old bucket
	for e := oldBucket.list.Front(); e != nil; e = e.Next() {
		contact := e.Value.(Contact)
		childIndex := bucketIndexFor(children, contact.ID)
		children[childIndex].list.PushFront(contact)
	}
	routingTable.buckets = append(routingTable.buckets[:idx], append(children, routingTable.buckets[idx+1:]...)...)
}

func (rt *RoutingTable) findLeafBucketIndex(id *KademliaID) int {
	for i, b := range rt.buckets {
		if b.containsID(id) {
			return i
		}
	}
	// Safety fallback (should never happen if prefixes are correct)
	return len(rt.buckets) - 1
}

func bucketIndexFor(children []*bucket, id *KademliaID) int {
	for i, b := range children {
		if b.containsID(id) {
			return i
		}
	}
	return len(children) - 1
}

func (routingTable *RoutingTable) ping(c Contact) bool {
	if routingTable.pingFn != nil {
		return routingTable.pingFn(c)
	}
	// default: be conservative (keep LRU). Better: return false to force replacement in tests.
	return true
}

// GetBit returns the bit (0/1) at absolute bit position pos (0 = MSB).
func (id *KademliaID) GetBit(pos int) int {
	if pos < 0 || pos >= IDLength*8 {
		return 0
	}
	byteIndex := pos / 8
	bitInByte := uint(7 - (pos % 8))
	if (id[byteIndex]>>bitInByte)&1 == 1 {
		return 1
	}
	return 0
}

// HasPrefix checks whether the first 'depth' bits of id equal prefix's first 'depth' bits.
func (id *KademliaID) HasPrefix(prefix *KademliaID, depth int) bool {
	for i := 0; i < depth; i++ {
		if id.GetBit(i) != prefix.GetBit(i) {
			return false
		}
	}
	return true
}

// CloneWithExtraBits returns a new KademliaID copying prefix and setting the next 'width' bits to 'value'.
func (prefix *KademliaID) CloneWithExtraBits(depth int, value int, width int) *KademliaID {
	// clone
	out := new(KademliaID)
	copy(out[:], prefix[:])

	// set bits [depth, depth+width)
	for w := 0; w < width; w++ {
		bitPos := depth + w
		bitVal := (value >> uint(width-1-w)) & 1
		byteIndex := bitPos / 8
		bitInByte := uint(7 - (bitPos % 8))
		if bitVal == 1 {
			out[byteIndex] |= (1 << bitInByte)
		} else {
			out[byteIndex] &^= (1 << bitInByte)
		}
	}
	return out
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
	distance := id.CalcDistance(routingTable.me.ID)
	bits := IDLength * 8
	for i := 0; i < IDLength; i++ {
		if distance[i] != 0 {
			for j := 0; j < 8; j++ {
				if (distance[i]>>uint8(7-j))&0x1 != 0 {
					bitIndex := i*8 + j
					closness := bits - bitIndex - 1

					return closness / BETA
				}
			}
		}
	}

	return len(routingTable.buckets) - 1
}

func (routingTable *RoutingTable) String() string {
	result := "Routing Table:\n"
	for i, bucket := range routingTable.buckets {
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
