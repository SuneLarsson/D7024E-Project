package main

import (
	"d7024e/kademlia"
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Starting Kademlia node...")

	id := kademlia.NewKademliaID("FFFFFFFF00000000000000000000000000000000")
	contact := kademlia.NewContact(id, "localhost:8000")
	fmt.Println("My contact info:", contact.String())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello! I am a Kademlia node.")
	})

	log.Println("Node is running and listening on port 8080.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
