package main

import (
	"d7024e/server"
	"fmt"
	"log"
	"os"
)

func main() {
	log.Println("Starting Kademlia network simulation...")

	// Read the bootstrap address from an environment variable.
	bootstrapAddress := os.Getenv("BOOTSTRAP_ADDRESS")

	fmt.Printf("Alpha = %s", os.Getenv("ALPHA"))
	// TODO: REMOVE WHEN KADEMLIA IS LISTENING
	serv := server.NewServer(server.DEFAULT_SOCKET, bootstrapAddress, 8000)
	serv.Listen()

}
