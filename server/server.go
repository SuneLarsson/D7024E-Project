package server

import (
	"bufio"
	"d7024e/kademlia"
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

type Server struct {
	socketPath string
	exitNode   bool
	mutExit    sync.RWMutex
	// storage          *storage.Storage
	node             *kademlia.Kademlia
	bootstrapAddress string
	port             int
}

func NewServer(sockPath string, bootstrapAddress string, port int) *Server {
	return &Server{
		socketPath:       sockPath,
		exitNode:         false,
		bootstrapAddress: bootstrapAddress,
		port:             port,
	}
}

// Starts begin listening for incoming messages
func (s *Server) Listen() {
	os.Remove(s.socketPath)

	// s.storage = storage.NewStorage()

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	node, err := kademlia.NewKademliaNode("0.0.0.0", s.port)
	if err != nil {
		log.Fatal("Failed to create Kademlia node:", err)
	}
	s.node = node
	log.Printf("Node created with ID: %s on address %s", s.node.Self.ID, s.node.Self.Address)
	//Start REST server

	if s.bootstrapAddress != "" {
		log.Printf("Attempting to join network via bootstrap node at %s", s.bootstrapAddress)
		go s.node.StartRESTServer(":8080")

		dummyContact := kademlia.NewContact(kademlia.NewRandomKademliaID(), s.bootstrapAddress)

		// Ping the bootstrap node. We only care about success or failure.
		var err error

		maxRetries := 5
		retryDelay := 2 * time.Second

		for i := 0; i < maxRetries; i++ {
			err = s.node.SendPing(&dummyContact)
			if err == nil {
				log.Printf("Successfully pinged bootstrap node. ")
				break
			}
			log.Printf("Failed to ping bootstrap node (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
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
		go s.node.StartRESTServer(":8081")
		log.Println("No bootstrap address provided. Starting as a bootstrap node.")
	}

	connCh := make(chan net.Conn)
	errCh := make(chan error)

	for {
		// 1. Check the exit condition
		s.mutExit.RLock()
		if s.exitNode {
			s.mutExit.RUnlock()
			break
		} else {
			s.mutExit.RUnlock()
		}

		// 2. Listen on the unix socket
		go func() {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- err
				return
			}
			connCh <- conn
		}()

		// 3. Wait 1 second for an interaction on one of the channels before going back to the loop
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
		case "forget":
			response := "This node will stop to refresh value assigned as " + splitRequest[1]
			s.node.Forget(splitRequest[1])
			reply(conn, response)
		case "get":
			// TODO: SEND BACK CONTACT
			if !s.node.IsValidKademliaID(splitRequest[1]) {
				reply(conn, "Invalid key")
				continue
			}
			var response *string
			_, response = s.node.LookupValue(splitRequest[1])
			if response != nil {
				reply(conn, *response)
			} else {
				reply(conn, "Value not found")
			}
		case "put":
			// TODO: CHANGE IF VALUE NOT STORED WELL
			var key string
			var result bool
			key, result = s.node.IterativeStore(splitRequest[1], true)
			if result {
				reply(conn, key)
			} else {
				reply(conn, "Value not stored")
			}
			// key, _ = s.node.IterativeStore(splitRequest[1])
			// reply(conn, key)
		case "routing":
			response := s.node.RoutingTable.String()

			reply(conn, response)
			reply(conn, "END")

		case "store":
			fmt.Println(s.node.RoutingTable.String())

			reply(conn, "END")
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
