package server

import (
	"bufio"
	"d7024e/kademlia"
	"d7024e/storage"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const SEPARATING_STRING string = ":"
const DEFAULT_SOCKET string = "/tmp/svc.sock"

var nbPortListenedTo int = 0

type Server struct {
	socketPath    string
	exitNode      bool
	mutExit       sync.RWMutex
	storage       *storage.Storage
	bootstrapNode kademlia.Kademlia
}

func NewServer(sockPath string) *Server {
	return &Server{socketPath: sockPath, exitNode: false}
}

// Starts begin listening for incoming messages
func (s Server) Listen() {
	os.Remove(s.socketPath)

	s.storage = storage.NewStorage()

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	// 1. Create the Bootstrap Node
	// This node starts first and acts as the entry point for others.
	bootstrapNode, err := kademlia.NewKademliaNode("127.0.0.1", 8000+nbPortListenedTo)
	nbPortListenedTo += 1
	if err != nil {
		log.Fatal("Failed to create bootstrap node:", err)
	}
	log.Printf("Bootstrap node created with ID: %s", bootstrapNode.Self.ID)

	// Give the bootstrap node a moment to start its listener
	time.Sleep(100 * time.Millisecond)

	/*// 2. Create a Second Node
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
	log.Println("Network is running. Press Ctrl+C to exit.")*/

	connCh := make(chan net.Conn)
	errCh := make(chan error)

	for {
		s.mutExit.RLock()
		if s.exitNode {
			s.mutExit.RUnlock()
			break
		} else {
			s.mutExit.RUnlock()
		}

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- err
				return
			}
			connCh <- conn
		}()

		select {
		case conn := <-connCh:
			go s.handleConnection(conn)
		case err := <-errCh:
			//TODO
			fmt.Println("Error on connection:", err)
		case <-time.After(1 * time.Second):

		}

	}

	ln.Close()
	os.Remove(s.socketPath)
}

// Handle the connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewScanner(conn)

	for reader.Scan() {
		request := strings.TrimSpace(reader.Text())

		splitRequest := strings.Split(request, SEPARATING_STRING)

		if len(splitRequest) < 1 {
			panic("Message shoud at least contain type of message")
		}

		switch splitRequest[0] {
		case "exit":
			s.mutExit.Lock()
			s.exitNode = true
			s.mutExit.Unlock()
		case "ping":
			reply(conn, "pong")
		case "get":
			// TODO: SEND BACK CONTACT
			var response *string
			_, response = s.bootstrapNode.LookupValue(splitRequest[1])
			reply(conn, *response)
		case "put":
			// TODO: CHANGE IF VALUE NOT STORED WELL
			var key string
			key, _ = s.bootstrapNode.IterativeStore(splitRequest[2])
			reply(conn, key)
		}
	}

	if err := reader.Err(); err != nil {
		fmt.Println("Connection closed with error:", err)
	} else {
		fmt.Println("Client disconnected.")
	}
}

// Sends a reply
func reply(conn net.Conn, reply string) {
	fmt.Fprintln(conn, reply)
}
