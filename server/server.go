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
	socketPath       string
	exitNode         bool
	mutExit          sync.RWMutex
	storage          *storage.Storage
	node             *kademlia.Kademlia
	bootstrapAddress string
}

func NewServer(sockPath string, bootstrapAddress string) *Server {
	return &Server{
		socketPath:       sockPath,
		exitNode:         false,
		bootstrapAddress: bootstrapAddress,
	}
}

// Starts begin listening for incoming messages
func (s *Server) Listen() {
	os.Remove(s.socketPath)

	s.storage = storage.NewStorage()

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	node, err := kademlia.NewKademliaNode("0.0.0.0", 8000)
	if err != nil {
		log.Fatal("Failed to create Kademlia node:", err)
	}
	s.node = node
	log.Printf("Node created with ID: %s on address %s", s.node.Self.ID, s.node.Self.Address)

	if s.bootstrapAddress != "" {
		log.Printf("Attempting to join network via bootstrap node at %s", s.bootstrapAddress)

		dummyContact := kademlia.NewContact(kademlia.NewRandomKademliaID(), s.bootstrapAddress)

		// Ping the bootstrap node. We only care about success or failure.
		var err error

		maxRetries := 5
		retryDelay := 20 * time.Second

		for i := 0; i < maxRetries; i++ {
			err = s.node.SendPing(&dummyContact)
			if err == nil {
				// Success!
				log.Printf("Successfully pinged bootstrap node. ")
				break
			}
			log.Printf("Failed to ping bootstrap node (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
		}

		if err != nil {
			// If it's still failing after all retries, then we exit.
			log.Fatalf("Could not connect to bootstrap node after %d attempts. Exiting.", maxRetries)
		}

		// Find the full contact info from our routing table.
		// Note: The bootstrap node should be the ONLY contact at this point.
		contacts := s.node.RoutingTable.FindClosestContacts(dummyContact.ID, 1)
		if len(contacts) < 1 {
			log.Fatal("Bootstrap contact not found in routing table after successful ping.")
		}

		bootstrapContact := contacts[0]
		log.Printf("Found bootstrap contact: %v", bootstrapContact)

		// Now, join the network using the real, complete contact info.
		s.node.JoinNetwork(&bootstrapContact)
	} else {
		log.Println("No bootstrap address provided. Starting as a bootstrap node.")
	}

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
			_, response = s.node.LookupValue(splitRequest[1])
			reply(conn, *response)
		case "put":
			// TODO: CHANGE IF VALUE NOT STORED WELL
			var key string
			key, _ = s.node.IterativeStore(splitRequest[2])
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
