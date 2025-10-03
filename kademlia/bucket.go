package kademlia

import (
	"container/list"
	"math/rand"
	"sync"
)

// bucket definition
// contains a List
type bucket struct {
	list   *list.List
	mu     sync.Mutex
	prefix *KademliaID
	depth  int
}

// newBucket returns a new instance of a bucket
func newBucket(prefix *KademliaID, depth int) *bucket {
	return &bucket{
		list:   list.New(),
		prefix: prefix,
		depth:  depth,
	}

}
func (bucket *bucket) containsID(id *KademliaID) bool {
	return bucket.prefix.HasPrefix(id, bucket.depth)
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
		if bucket.list.Len() < bucketSize {
			bucket.list.PushFront(contact)
		}
	} else {
		bucket.list.MoveToFront(element)
	}
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
