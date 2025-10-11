package server

import (
	"bufio"
	"d7024e/kademlia"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const SEPARATING_STRING string = ":"

var Default_socket string = filepath.Join(os.TempDir(), "svc_app.sock")

const ERR_NOMESSAGE string = "Message should at least contain type of message"
const ERR_INVALIDSOCKET string = "Error listening on socket"
const ERR_NODECREATIONFAILURE string = "Error creating kademlia node"

type Server struct {
	socketPath string
	exitNode   bool
	mutExit    sync.RWMutex
	// storage          *storage.Storage
	node             *kademlia.Kademlia
	bootstrapAddress string
	port             int
	restPort         int
}

func NewServer(sockPath string, bootstrapAddress string, port int) *Server {
	restPortStr := os.Getenv("REST_PORT")
	if restPortStr == "" {
		restPortStr = "8081" // Default port
	}

	restPort, err := strconv.Atoi(restPortStr)
	if err != nil {
		log.Fatalf("Invalid REST_PORT value: %v", err)
	}

	return &Server{
		socketPath:       sockPath,
		exitNode:         false,
		bootstrapAddress: bootstrapAddress,
		port:             port,
		restPort:         restPort,
	}
}

// Changing the rest port (ONLY FOR TESTING PURPOSES)
func (s *Server) SetRestPort(port int) {
	s.restPort = port
}

// Starts begin listening for incoming messages
func (s *Server) Listen() {
	os.Remove(s.socketPath)

	// s.storage = storage.NewStorage()
	var ln net.Listener
	var err error

	ln, err = net.Listen("unix", s.socketPath)
	if err != nil {
		//fmt.Println(err)
		panic(ERR_INVALIDSOCKET)
	}

	node, err := kademlia.NewKademliaNode("0.0.0.0", s.port)
	if err != nil {
		fmt.Println(ERR_NODECREATIONFAILURE, ":", err)
		panic(ERR_NODECREATIONFAILURE)
	}
	s.node = node
	log.Printf("Node created with ID: %s on address %s", s.node.Self.ID, s.node.Self.Address)
	//Start REST server

	if s.bootstrapAddress != "" {
		log.Printf("Attempting to join network via bootstrap node at %s", s.bootstrapAddress)
		restAddr := fmt.Sprintf(":%d", s.restPort)

		go s.node.StartRESTServer(restAddr)

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
			fmt.Println("Could not connect to bootstrap node after", maxRetries, "attempts. Exiting.")
			panic(ERR_NODECREATIONFAILURE)
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
		restAddr := fmt.Sprintf(":%d", s.restPort)

		go s.node.StartRESTServer(restAddr)
		log.Println("No bootstrap address provided. Starting as a bootstrap node.")
	}

	connCh := make(chan net.Conn)
	errCh := make(chan error)

	// 1. Goroutine dedicated to accepting connections
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- err
				return // exit goroutine when listener is closed
			}
			connCh <- conn
		}
	}()

	// 2. Main loop
	for {
		s.mutExit.RLock()
		exit := s.exitNode
		s.mutExit.RUnlock()
		if exit {
			ln.Close() // this will unblock ln.Accept() above
			break
		}

		select {
		case conn := <-connCh:
			go s.handleConnection(conn)
		case err := <-errCh:
			fmt.Println("Error on connection:", err)
		case <-time.After(1 * time.Second):
			// periodic tick — could be used to check conditions, etc.
		}
	}

	node.Shutdown()
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

		if len(splitRequest) >= 1 {

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
				if !s.node.IsValidKademliaID(splitRequest[1]) {
					reply(conn, "Invalid key")
					continue
				}
				var response *string
				_, response = s.node.LookupValue(splitRequest[1])
				if response != nil && *response != "" {
					reply(conn, *response)
				} else {
					reply(conn, "Value not found")
				}
			case "put":
				var key string
				var result bool
				key, result = s.node.IterativeStore(splitRequest[1], true)
				if result {
					reply(conn, key)
				} else {
					reply(conn, "Value not stored")
				}
			case "routing":
				response := s.node.RoutingTable.String()

				reply(conn, response)
				reply(conn, "END")

			case "store":
				fmt.Println(s.node.RoutingTable.String())

				reply(conn, "END")
			}
		}
	}

	fmt.Println("Client disconnected.")
}

// Sends a reply
func reply(conn net.Conn, reply string) {
	fmt.Fprintln(conn, reply)
}
