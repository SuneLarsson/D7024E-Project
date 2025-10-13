package kademlia

import (
	"container/list"
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// bucket definition
// contains a List
type bucket struct {
	list   *list.List
	mu     sync.Mutex
	prefix string
	depth  int
	b      int
}

// newBucket returns a new instance of a bucket
func newBucket(prefix string, depth int) *bucket {
	bucket := &bucket{}
	bucket.list = list.New()
	bucket.prefix = prefix
	bucket.depth = depth
	configMutex.Lock()
	bucket.b = BETA
	configMutex.Unlock()
	return bucket
}

// AddContact adds the Contact to the front of the bucket
// or moves it to the front of the bucket if it already existed
func (bucket *bucket) AddContact(contact Contact) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	var element *list.Element
	for e := bucket.list.Front(); e != nil; e = e.Next() {
		nodeID := e.Value.(Contact).ID

		if (contact).ID.Equals(nodeID) {
			element = e
		}
	}

	if element == nil {
		configMutex.Lock()
		defer configMutex.Unlock()
		if bucket.list.Len() < K {
			fmt.Println("Pushing", contact.Address, "to FRONT")
			bucket.list.PushFront(contact)
		}
	} else {
		bucket.list.MoveToFront(element)
	}
}

// CanAddContact verifies whether a contact can be added to the current bucket
// If it cannot but the bucket can be split, then it says so
func (bucket *bucket) CanAddContact(contact Contact, kademlia *Kademlia) (bool, bool) {
	var canAdd bool = false
	var canSplit bool = false

	if bucket.ContainsID(contact.ID) {
		for e := bucket.list.Front(); e != nil; e = e.Next() {
			nodeID := e.Value.(Contact).ID

			if contact.ID.Equals(nodeID) {
				canAdd = true
			}
		}

		// Must be < K because <= K would lead to K+1 sized bucket
		configMutex.Lock()
		if !canAdd && bucket.list.Len() < K {
			canAdd = true
		}
		configMutex.Unlock()

		if !canAdd && (bucket.ContainsID(kademlia.Self.ID) || bucket.depth%bucket.b != 0) {
			canSplit = true
		}

		if !canAdd && !canSplit && bucket.list.Len() == K {
			backElement := bucket.list.Back()
			otherNode := backElement.Value.(Contact)
			bucket.mu.Lock()
			err := kademlia.SendPing(&otherNode)
			if err == nil {
				bucket.list.MoveToFront(backElement)
			} else {
				bucket.list.Remove(backElement)
				canAdd = true
			}
			bucket.mu.Unlock()
		}
	}
	return canAdd, canSplit
}

// ContainsID checks if the KademliaID has the necessary prefix for the bucket
func (bucket *bucket) ContainsID(id *KademliaID) bool {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Normalize prefix (trim spaces/newlines)
	prefix := strings.TrimSpace(bucket.prefix)
	depth := bucket.depth
	binID := strings.TrimSpace(id.BinaryString())

	// Sanity checks
	if len(binID) < depth {
		// The binary ID is shorter than the depth we're checking — cannot match
		return false
	}

	if len(prefix) != depth {
		// Mismatch between prefix length and depth — usually a bucket setup issue
		// You can either return false or enforce correction
		// For safety, compare up to the shorter of the two
		minLen := len(prefix)
		if depth < minLen {
			minLen = depth
		}
		return binID[:minLen] == prefix[:minLen]
	}

	// Main prefix match check
	matches := binID[:depth] == prefix

	return matches
}

// IsPresent checks if the KademliaID is present in the bucket
func (bucket *bucket) IsPresent(id *KademliaID) bool {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	for e := bucket.list.Back(); e != nil; e = e.Prev() {
		nodeID := e.Value.(Contact).ID
		if nodeID.String() == id.String() {
			return true
		}
	}
	return false
}

// SplitBucket splits a bucket into two different buckets by creating a new one and returning both
func (bucket *bucket) SplitBucket() (*bucket, *bucket) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	bucket1 := newBucket(bucket.prefix+"1", bucket.depth+1)
	bucket1.b = bucket.b
	bucket0 := newBucket(bucket.prefix+"0", bucket.depth+1)
	bucket0.b = bucket.b
	bucket.depth++
	for e := bucket.list.Back(); e != nil; e = e.Prev() {
		nodeID := e.Value.(Contact).ID
		if bucket1.ContainsID(nodeID) {
			bucket1.list.PushFront(e.Value.(Contact))
		} else {
			bucket0.list.PushFront(e.Value.(Contact))
		}
	}
	return bucket1, bucket0
}

// GetContactAndCalcDistance returns an array of Contacts where
// the distance has already been calculated
func (bucket *bucket) GetContactAndCalcDistance(target *KademliaID) []Contact {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	var contacts []Contact

	for elt := bucket.list.Front(); elt != nil; elt = elt.Next() {
		contact := elt.Value.(Contact)
		contact.CalcDistance(target)
		contacts = append(contacts, contact)
	}

	return contacts
}

// Len return the size of the bucket
func (bucket *bucket) Len() int {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	return bucket.list.Len()
}

// Refresh bucket the node selects a random number in that range and does a refresh, an iterativeFindNode using that number as key.
func (bucket *bucket) getContactForBucketRefresh() Contact {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	if bucket.list.Len() == 0 {
		return Contact{ID: nil} // Return a contact with a nil ID
	}
	randomIndex := rand.Intn(bucket.list.Len())
	element := bucket.list.Front()
	for i := 0; i < randomIndex; i++ {
		element = element.Next()
	}
	return element.Value.(Contact)
}
