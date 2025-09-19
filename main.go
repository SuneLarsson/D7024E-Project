package main

import (
	"d7024e/server"
	"fmt"
	"os"
)

func main() {
	fmt.Println("Starting Kademlia network simulation...")

	// Read the bootstrap address from an environment variable.
	bootstrapAddress := os.Getenv("BOOTSTRAP_ADDRESS")

	// TODO: REMOVE WHEN KADEMLIA IS LISTENING
	serv := server.NewServer(server.DEFAULT_SOCKET, bootstrapAddress)
	serv.Listen()

}
