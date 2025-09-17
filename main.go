package main

import (
	"d7024e/server"
	"fmt"
)

func main() {
	fmt.Println("Starting Kademlia network simulation...")

	// TODO: REMOVE WHEN KADEMLIA IS LISTENING
	serv := server.NewServer(server.DEFAULT_SOCKET)
	serv.Listen()

}
