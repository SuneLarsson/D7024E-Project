package kademlia

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

func (kademlia *Kademlia) LookupNode(target string) []Contact {
	targetId := NewKademliaID(target)
	return kademlia.IterativeFindNode(targetId, ALPHA, K)
}

func (kademlia *Kademlia) LookupValue(target string) ([]Contact, *string) {
	if !kademlia.IsValidKademliaID(target) {
		return nil, nil
	}
	targetId := NewKademliaID(target)
	// TODO if the value exists in the local datastore should we return it directly?
	dataItem, exists := kademlia.DataStore.Get(targetId.String())
	if exists {
		return nil, &dataItem
	}
	return kademlia.IterativeFindValue(targetId, ALPHA, K)
}

func idsOf(contacts []Contact) []string {
	ids := make([]string, len(contacts))
	for i, c := range contacts {
		ids[i] = c.ID.String()
	}
	return ids
}

func (kademlia *Kademlia) IterativeFindValue(target *KademliaID, alpha int, kSize int) ([]Contact, *string) {
	type findValueResponse struct {
		from     *Contact
		contacts []Contact
		value    *string
	}

	probed := make(map[string]bool)
	candidates := &ContactCandidates{}
	shortlist := kademlia.RoutingTable.FindClosestContacts(target, alpha)
	candidates.Append(shortlist)
	candidates.Sort()

	queried := make(map[string]bool)

	var nodeWithoutValue *Contact = nil

	var closestSoFar *Contact

	if candidates.Len() > 0 {
		closestSoFar = &candidates.contacts[0]
	}

	queried[kademlia.Self.ID.String()] = true

	for {
		nodesToQuery := candidates.pickAlpha(queried, alpha)

		if len(nodesToQuery) == 0 {
			break
		}

		isStalled := closestSoFar != nil && !nodesToQuery[0].Less(closestSoFar)
		if isStalled {
			nodesToQuery = candidates.pickAlpha(queried, kSize)
			if len(nodesToQuery) == 0 {
				break
			}
		}

		responseChan := make(chan findValueResponse, len(nodesToQuery))
		// nbAwaitedAnswer := len(nodesToQuery)
		for _, c := range nodesToQuery {
			queried[c.ID.String()] = true
			go func(contact Contact) {
				// Use the FindValue RPC instead of FindNode
				contacts, found, val := kademlia.FindValue(&contact, target)
				if found {
					responseChan <- findValueResponse{from: &contact, contacts: nil, value: val}
				} else {
					responseChan <- findValueResponse{from: &contact, contacts: contacts, value: nil}
				}
			}(c)
		}

		roundTimeout := time.After(3 * time.Second)
		var valueFound *string = nil
		for i := 0; i < len(nodesToQuery); i++ {
			select {
			case resp := <-responseChan:
				if resp.value != nil {
					valueFound = resp.value
					hash := sha1.Sum([]byte(*valueFound))
					key := hex.EncodeToString(hash[:])

					// Launch the Store call in a separate goroutine and move on.
					go func(node *Contact, val string, k string) {
						if node != nil {
							kademlia.Store(node, val, k, false)
						}
					}(nodeWithoutValue, *valueFound, key)
					return nil, valueFound
				}

				if resp.contacts != nil {
					probed[resp.from.ID.String()] = true
					candidates.mergeAndSort(resp.contacts, target, kSize)
					contactThatReplied := resp.from
					contactThatReplied.CalcDistance(target)
					if nodeWithoutValue == nil || contactThatReplied.Less(nodeWithoutValue) {
						nodeWithoutValue = contactThatReplied
					}
				}

			case <-roundTimeout:
				goto endRound
			}
		}

	endRound:

		if candidates.Len() == 0 {
			break
		}

		newClosestNode := &candidates.contacts[0]
		if !newClosestNode.Less(closestSoFar) && isStalled {
			break
		}
		closestSoFar = newClosestNode

		probedCount := 0
		for i := 0; i < candidates.Len() && i < kSize; i++ {
			if probed[candidates.contacts[i].ID.String()] {
				probedCount++
			}
		}
		if probedCount >= kSize {
			break
		}

	}
	if candidates.Len() < kSize {
		return candidates.GetContacts(candidates.Len()), nil
	}
	return candidates.GetContacts(kSize), nil
}

func (kademlia *Kademlia) IterativeStore(value string, originalUploader bool) (string, bool) {
	//1. Hash the value to get the key
	dataToHash := []byte(value)
	hash := sha1.Sum(dataToHash)
	key := NewKademliaID(hex.EncodeToString(hash[:]))

	kademlia.DataStore.Put(key.String(), value, true, true)

	//2. Find the k closest nodes to the key
	closest := kademlia.IterativeFindNode(key, ALPHA, K)
	log.Printf("Found %d closest nodes to store the value: %v\n", len(closest), idsOf(closest))

	//3. Send STORE RPCs to those nodes
	successCount := 0
	chStore := make(chan bool, len(closest))

	for _, contact := range closest {
		go func(c Contact) {
			chStore <- kademlia.Store(&c, value, key.String(), originalUploader)
		}(contact)
	}

	for range closest {
		if <-chStore {
			successCount++
		}
	}

	// If at least one STORE was successful, consider it a success
	// and print the number of successful stores
	// Otherwise, print a failure message
	if successCount > 0 && originalUploader {
		log.Printf("Successfully stored value on %d nodes\n", successCount)
		go func(key *KademliaID) {
			ticker := time.NewTicker(time.Duration(tRepublish))
			defer ticker.Stop()
			forgetChan := make(chan string)
			kademlia.keyMutex.Lock()
			kademlia.keyStore[key.String()] = forgetChan
			kademlia.keyMutex.Unlock()
			keepGoing := true
			for keepGoing {
				select {
				case <-ticker.C:
					kademlia.IterativeRefresh(key, 3)
				case <-forgetChan:
					keepGoing = false
				}
			}
			fmt.Println("No more refreshing the value that has key", key.String())
		}(key)
	} else if !originalUploader {
		log.Println("Not Original uploader")
	} else {
		log.Println("Failed to store value on any node")
	}

	//4. If a node does not respond, find a replacement node and send STORE to it // Optional
	return key.String(), successCount > 0
}

func (kademlia *Kademlia) IterativeRefresh(key *KademliaID, kSize int) {
	contact := kademlia.RoutingTable.FindClosestContacts(key, kSize)
	for _, c := range contact {
		go kademlia.Refresh(&c, key.String())
	}
}

type findNodeResponse struct {
	from     Contact
	contacts []Contact
}

func (kademlia *Kademlia) IterativeFindNode(target *KademliaID, alpha int, kSize int) []Contact {
	probed := make(map[string]bool)

	candidates := &ContactCandidates{}
	shortlist := kademlia.RoutingTable.FindClosestContacts(target, alpha)
	candidates.Append(shortlist)
	candidates.Sort()

	var closestSoFar *Contact

	if candidates.Len() > 0 {
		closestSoFar = &candidates.contacts[0]
	}

	queried := make(map[string]bool)
	queried[kademlia.Self.ID.String()] = true
	for {
		nodesToQuery := candidates.pickAlpha(queried, alpha)

		if len(nodesToQuery) == 0 {
			break
		}

		isStalled := closestSoFar != nil && !nodesToQuery[0].Less(closestSoFar)
		if isStalled {
			nodesToQuery = candidates.pickAlpha(queried, kSize)
			if len(nodesToQuery) == 0 {
				break
			}
		}

		responseChan := make(chan findNodeResponse, len(nodesToQuery))
		for _, c := range nodesToQuery {
			queried[c.ID.String()] = true
			go func(contact Contact) {
				contacts, result, _ := kademlia.FindNode(&contact, target)
				if result {
					responseChan <- findNodeResponse{from: contact, contacts: contacts}
				} else {
					responseChan <- findNodeResponse{from: contact, contacts: nil}
				}
			}(c)
		}

		responses := 0

		roundTimeout := time.After(3 * time.Second)
		for i := 0; i < len(nodesToQuery); i++ {
			select {
			case resp := <-responseChan:
				responses++
				if resp.contacts != nil {
					probed[resp.from.ID.String()] = true
					candidates.mergeAndSort(resp.contacts, target, kSize)
				}
			case <-roundTimeout:
				goto endRound
			}
		}
	endRound:

		newClosestNode := &candidates.contacts[0]
		if !newClosestNode.Less(closestSoFar) && isStalled {
			// If the search was stalled AND this round found no one better, we are done.
			break
		}
		closestSoFar = newClosestNode

		probedCount := 0
		for i := 0; i < candidates.Len() && i < kSize; i++ {
			if probed[candidates.contacts[i].ID.String()] {
				probedCount++
			}
		}
		if probedCount >= kSize {
			break
		}

	}

	if candidates.Len() < kSize {
		// fmt.Printf("Number of Candiates %v\n", candidates.Len())
		return candidates.GetContacts(candidates.Len())
	}
	return candidates.GetContacts(kSize)
}

func (c *ContactCandidates) pickAlpha(queried map[string]bool, alpha int) []Contact {
	toQuery := []Contact{}
	for _, contact := range c.contacts {
		if len(toQuery) >= alpha {
			break
		}
		if !queried[contact.ID.String()] {
			toQuery = append(toQuery, contact)
		}
	}
	return toQuery
}

func (c *ContactCandidates) mergeAndSort(newContacts []Contact, target *KademliaID, kSize int) bool {
	progress := false
	for _, nc := range newContacts {
		nc.CalcDistance(target)
		if !containsContact(c.contacts, nc) {
			c.Append([]Contact{nc})
			progress = true
		}
	}

	c.SortWithTarget(target)

	if c.Len() > kSize {
		c.contacts = c.GetContacts(kSize)
	}
	return progress
}

func (c *ContactCandidates) SortWithTarget(target *KademliaID) {
	for i := range c.contacts {
		if c.contacts[i].distance == nil {
			c.contacts[i].CalcDistance(target)
		}
	}
	c.Sort()
}

func containsContact(list []Contact, c Contact) bool {
	for _, x := range list {
		if x.ID.Equals(c.ID) {
			return true
		}
	}
	return false
}
