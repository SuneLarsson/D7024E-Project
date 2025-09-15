package main

import (
	"d7024e/kademlia"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("Starting Kademlia network simulation...")

	// 1. Create the Bootstrap Node
	// This node starts first and acts as the entry point for others.
	bootstrapNode, err := kademlia.NewKademliaNode("127.0.0.1", 8000)
	if err != nil {
		log.Fatal("Failed to create bootstrap node:", err)
	}
	log.Printf("Bootstrap node created with ID: %s", bootstrapNode.Self.ID)

	// Give the bootstrap node a moment to start its listener
	time.Sleep(100 * time.Millisecond)

	// 2. Create a Second Node
	joiningNode, err := kademlia.NewKademliaNode("127.0.0.1", 8001)
	if err != nil {
		log.Fatal("Failed to create second node:", err)
	}
	log.Printf("Joining node created with ID: %s", joiningNode.Self.ID)

	// 3. Join the Network
	// The joining node needs the contact info of the bootstrap node to connect.
	bootstrapContact := kademlia.NewContact(bootstrapNode.Self.ID, bootstrapNode.Self.Address)
	log.Printf("Joining node is connecting to bootstrap node at %s", bootstrapContact.Address)
	joiningNode.JoinNetwork(&bootstrapContact)

	// Give the join process a moment to run
	time.Sleep(1 * time.Second)

	// 4. Perform a Lookup
	// Have the joining node look for the bootstrap node's ID.
	log.Printf("Joining node is now looking for bootstrap node's ID: %s", bootstrapNode.Self.ID)
	foundContacts := joiningNode.IterativeFindNode(bootstrapNode.Self.ID)

	// 5. Print the results
	if len(foundContacts) > 0 {
		log.Println("Lookup successful! Found contacts:")
		for _, contact := range foundContacts {
			fmt.Printf("  - Contact: %+v, Distance: %s\n", contact, contact.ID.CalcDistance(bootstrapNode.Self.ID))
		}
	} else {
		log.Println("Lookup failed: No contacts found.")
	}

	// 6. Keep the program running so the nodes can listen in the background
	log.Println("Network is running. Press Ctrl+C to exit.")
	select {} // Block forever
}
