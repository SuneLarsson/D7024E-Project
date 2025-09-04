package main

import (
	"d7024e/kademlia"
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Starting Kademlia node...")

	// This is the code from your original main.go
	id := kademlia.NewKademliaID("FFFFFFFF00000000000000000000000000000000")
	contact := kademlia.NewContact(id, "localhost:8000")
	fmt.Println("My contact info:", contact.String())

	// To keep the container running, we start a simple web server.
	// This is a common pattern for services that need to stay alive.
	// It will listen for HTTP requests on port 8080.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello! I am a Kademlia node.")
	})

	// The log.Fatal will cause the program to exit if the server fails to start.
	log.Println("Node is running and listening on port 8080.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
