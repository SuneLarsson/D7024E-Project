package kademlia

import (
	"container/list"
	"math/rand"
)

// bucket definition
// contains a List
type bucket struct {
	list *list.List
}

// newBucket returns a new instance of a bucket
func newBucket() *bucket {
	bucket := &bucket{}
	bucket.list = list.New()
	return bucket
}

// AddContact adds the Contact to the front of the bucket
// or moves it to the front of the bucket if it already existed
func (bucket *bucket) AddContact(contact Contact) {
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
	return bucket.list.Len()
}

// Refresh bucket the node selects a random number in that range and does a refresh, an iterativeFindNode using that number as key.
func (bucket *bucket) getContactForBucketRefresh() Contact {
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
