package main

import (
	"d7024e/kademlia"
	"d7024e/storage"
	"fmt"
)

func main() {
	fmt.Println("Starting Kademlia node...")

	// This is the code from your original main.go
	id := kademlia.NewKademliaID("FFFFFFFF00000000000000000000000000000000")
	contact := kademlia.NewContact(id, "localhost:8000")
	fmt.Println("My contact info:", contact.String())

	storage := storage.NewServer(storage.DEFAULT_SOCKET)

	storage.Listen()

	fmt.Println("End of Node")

}
