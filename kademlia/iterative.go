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
	return kademlia.IterativeFindNode(targetId, ALPHA, K, false)
}

func (kademlia *Kademlia) LookupValue(target string) ([]Contact, *string) {
	if !kademlia.IsValidKademliaID(target) {
		return nil, nil
	}
	targetId := NewKademliaID(target)
	// TODO if the value exists in the local datastore should we return it directly?
	// dataItem, exists := kademlia.DataStore.Get(targetId.String())
	// if exists {
	// 	return nil, &dataItem
	// }
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
	candidates := &ContactCandidates{}
	shortlist := kademlia.RoutingTable.FindClosestContacts(target, alpha)
	candidates.Append(shortlist)
	queried := make(map[string]bool)

	var nodeWithoutValue *Contact = nil

	for {
		nodesToQuery := candidates.pickAlpha(queried, alpha)

		if len(nodesToQuery) == 0 {
			fmt.Println("[IterativeFindValue] No more nodes to query, stopping")
			break
		}
		fmt.Printf("[IterativeFindValue] Querying %d nodes: %v\n", len(nodesToQuery), idsOf(nodesToQuery))
		type findValueResponse struct {
			from     *Contact
			contacts []Contact
			value    *string
		}

		responseChan := make(chan findValueResponse, len(nodesToQuery))
		nbAwaitedAnswer := len(nodesToQuery)
		for _, c := range nodesToQuery {
			queried[c.ID.String()] = true
			if kademlia.Self.ID.Equals(c.ID) {
				nbAwaitedAnswer--
				continue
			}
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
		for i := 0; i < nbAwaitedAnswer; i++ {
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
	if candidates.Len() < kSize {
		return candidates.GetContacts(candidates.Len()), nil
	}
	return candidates.GetContacts(kSize), nil
}

func (kademlia *Kademlia) IterativeStore(value string) (string, bool) {
	//1. Hash the value to get the key
	dataToHash := []byte(value)
	hash := sha1.Sum(dataToHash)
	key := NewKademliaID(hex.EncodeToString(hash[:]))
	// log.Printf("Storing value with key %s\n", key)
	// log.Printf("Current routing table: %v\n", *kademlia.RoutingTable)
	//2. Find the k closest nodes to the key
	closest := kademlia.IterativeFindNode(key, ALPHA, K, false)
	log.Printf("Found %d closest nodes to store the value: %v\n", len(closest), idsOf(closest))
	// closest := kademlia.IterativeFindNode(key)
	//3. Send STORE RPCs to those nodes
	successCount := 0
	chStore := make(chan bool, len(closest))

	for _, contact := range closest {
		go func(c Contact) {
			chStore <- kademlia.Store(&c, value, key.String())
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
	if successCount > 0 {
		log.Printf("Successfully stored value on %d nodes\n", successCount)
		go func(key *KademliaID) {
			ticker := time.NewTicker(12 * time.Hour)
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

func (kademlia *Kademlia) IterativeFindNode(target *KademliaID, alpha int, kSize int, log bool) []Contact {
	probed := make(map[string]bool)

	candidates := &ContactCandidates{}
	shortlist := kademlia.RoutingTable.FindClosestContacts(target, alpha)
	candidates.Append(shortlist)
	candidates.Sort()

	var closestSoFar *Contact
	// kademlia.Self.CalcDistance(target)
	// closestSoFar = &kademlia.Self

	// if candidates.Len() > 0 {
	// 	bestFromShortlist := &candidates.contacts[0]
	// 	if bestFromShortlist.Less(closestSoFar) {
	// 		closestSoFar = bestFromShortlist
	// 	}
	// }

	if candidates.Len() > 0 {
		closestSoFar = &candidates.contacts[0]
	}

	queried := make(map[string]bool)
	queried[kademlia.Self.ID.String()] = true
	// probed[kademlia.Self.ID.String()] = true

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
		if log {
			fmt.Printf("[IterativeFindNode] Querying %d nodes: %v\n", len(nodesToQuery), idsOf(nodesToQuery))
		}

		// nbAwaitedAnswer := len(nodesToQuery)
		responseChan := make(chan findNodeResponse, len(nodesToQuery))
		for _, c := range nodesToQuery {
			queried[c.ID.String()] = true
			// contacts, _, _ := kademlia.FindNode(&c, target)
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

		// progress := false
		roundTimeout := time.After(3 * time.Second)
		for i := 0; i < len(nodesToQuery); i++ {
			select {
			case resp := <-responseChan:
				responses++
				if resp.contacts != nil {
					probed[resp.from.ID.String()] = true
					candidates.mergeAndSort(resp.contacts, target, kSize)
				}
				// if candidates.mergeAndSort(newContacts, target, kSize) {
				// 	progress = true
				// }
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

		probedCount := 0
		for i := 0; i < candidates.Len() && i < kSize; i++ {
			if probed[candidates.contacts[i].ID.String()] {
				probedCount++
			}
		}
		if probedCount >= kSize {
			// We have confirmed the K best nodes are active.
			break
		}
		// if candidates.Len() > 0 {
		// 	closestDistance := candidates.contacts[0]
		// 	if closestSoFar == nil || closestDistance.Less(closestSoFar) {
		// 		closestSoFar = &closestDistance
		// 	} else if !progress {
		// 		break
		// 	}
		// }

		// if !progress {
		// 	break
		// }
		if log {
			fmt.Printf("Canditates, %v\n", candidates.contacts)
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
