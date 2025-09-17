package kademlia

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

func (kademlia *Kademlia) LookupNode(target string) []Contact {
	targetId := NewKademliaID(target)
	return kademlia.IterativeFindNode(targetId)
}

func (kademlia *Kademlia) LookupValue(target string) ([]Contact, *string) {
	targetId := NewKademliaID(target)
	return kademlia.IterativeFindValue(targetId)
}

func (kademlia *Kademlia) IterativeFindValue(target *KademliaID) ([]Contact, *string) {
	candidates := &ContactCandidates{}
	shortlist := kademlia.RoutingTable.FindClosestContacts(target, alpha)
	candidates.Append(shortlist)
	queried := make(map[string]bool)

	var nodeWithoutValue *Contact = nil

	for {
		nodesToQuery := candidates.pickAlpha(queried, alpha)

		if len(nodesToQuery) == 0 {
			break
		}

		type findValueResponse struct {
			from     *Contact
			contacts []Contact
			value    *string
		}

		responseChan := make(chan findValueResponse, len(nodesToQuery))
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

		var roundResponses []findValueResponse
		for i := 0; i < len(nodesToQuery); i++ {
			roundResponses = append(roundResponses, <-responseChan)
		}

		progress := false
		var valueFound *string = nil

		for _, resp := range roundResponses {
			if resp.value != nil {
				valueFound = resp.value
			} else {
				// This node did NOT have the value. It's a candidate for caching.
				resp.from.CalcDistance(target)
				if nodeWithoutValue == nil || resp.from.Less(nodeWithoutValue) {
					nodeCopy := *resp.from
					nodeWithoutValue = &nodeCopy
				}

				// Merge the new contacts from the response.
				if candidates.mergeAndSort(resp.contacts, target, kSize) {
					progress = true
				}
			}
		}

		if valueFound != nil {
			hash := sha1.Sum([]byte(*valueFound))
			key := hex.EncodeToString(hash[:])

			// Launch the Store call in a separate goroutine and move on.
			go func(node *Contact, val string, k string) {
				if node != nil {
					kademlia.Store(node, val, k)
				}
			}(nodeWithoutValue, *valueFound, key)

			// The function returns IMMEDIATELY without waiting for the Store to finish.
			return nil, valueFound
		}

		// If no progress was made, stall and exit.
		if !progress {
			break
		}
	}
	return candidates.GetContacts(kSize), nil
}

func (kademlia *Kademlia) IterativeStore(value string) (string, bool) {
	//1. Hash the value to get the key
	dataToHash := []byte(value)
	hash := sha1.Sum(dataToHash)
	key := NewKademliaID(hex.EncodeToString(hash[:]))

	//2. Find the k closest nodes to the key
	closest := kademlia.IterativeFindNode(key)
	// closest := kademlia.IterativeFindNode(key)
	//3. Send STORE RPCs to those nodes
	successCount := 0

	for _, contact := range closest {
		result := kademlia.Store(&contact, value, key.String())
		if result {
			successCount++
		}
	}
	// If at least one STORE was successful, consider it a success
	// and print the number of successful stores
	// Otherwise, print a failure message
	if successCount > 0 {
		fmt.Printf("Successfully stored value on %d nodes\n", successCount)
	} else {
		fmt.Println("Failed to store value on any node")
	}

	//4. If a node does not respond, find a replacement node and send STORE to it // Optional
	return key.String(), successCount > 0
}

func (kademlia *Kademlia) IterativeFindNode(target *KademliaID) []Contact {
	candidates := &ContactCandidates{}
	shortlist := kademlia.RoutingTable.FindClosestContacts(target, alpha)
	candidates.Append(shortlist)
	closestSoFar := &Contact{}
	queried := make(map[string]bool)

	for {
		nodesToQuery := candidates.pickAlpha(queried, alpha)

		if len(nodesToQuery) == 0 {
			break
		}

		responseChan := make(chan []Contact, len(nodesToQuery))
		for _, c := range nodesToQuery {
			queried[c.ID.String()] = true
			// contacts, _, _ := kademlia.FindNode(&c, target)
			go func(contact Contact) {
				contacts, _, _ := kademlia.FindNode(&contact, target)
				responseChan <- contacts
			}(c)
		}

		progress := false
		for i := 0; i < len(nodesToQuery); i++ {
			newContacts := <-responseChan
			if candidates.mergeAndSort(newContacts, target, kSize) {
				progress = true
			}
		}

		if candidates.Len() > 0 {
			closestDistance := candidates.contacts[0]
			if closestSoFar == nil || closestDistance.Less(closestSoFar) {
				closestSoFar = &closestDistance
			} else if !progress {
				break
			}
		}

		if !progress {
			break
		}

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

	c.Sort()
	if c.Len() > kSize {
		c.contacts = c.GetContacts(kSize)
	}
	return progress
}

func containsContact(list []Contact, c Contact) bool {
	for _, x := range list {
		if x.ID.Equals(c.ID) {
			return true
		}
	}
	return false
}
