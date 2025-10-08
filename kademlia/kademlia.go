package kademlia

import (
	"d7024e/storage"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// const kSize = 20 // Bucket size
// const alpha = 3  // Concurrency

// Kademlia structure

type Kademlia struct {
	Self         Contact
	Network      NetworkAPI
	RoutingTable *RoutingTable
	mapManagerCh chan MapRequest
	DataStore    storage.Storage
	keyStore     map[string]chan string
	keyMutex     sync.Mutex
	httpServer   *http.Server
	alpha        int
	beta         int
	k            int
	ttl          time.Duration
}

type MapRequest struct {
	rpcID        KademliaID
	responseChan chan Message
	register     bool
	responseMsg  Message
}

type DataItem struct {
	value      string
	timeToLive time.Time
}

func NewKademliaNode(ip string, port int) (*Kademlia, error) {
	// 1. Resolve the listening address (using "0.0.0.0" is correct here)
	listenAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", "0.0.0.0", port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return nil, err
	}

	outboundIP, err := getOutboundIP()
	if err != nil {
		log.Printf("Could not determine outbound IP, falling back to localhost: %v", err)
		outboundIP = "127.0.0.1" // Fallback for local testing
	}

	// 3. Create the Contact with the CORRECT, public address
	contact := Contact{
		ID:       NewRandomKademliaID(),
		Address:  fmt.Sprintf("%s:%d", outboundIP, port), // Use the discovered IP
		distance: nil,
	}

	routingtable := NewRoutingTable(contact)

	ttl := time.Duration(TTL) * time.Second

	kademlia := &Kademlia{
		Self:         contact,
		RoutingTable: routingtable,
		mapManagerCh: make(chan MapRequest),
		DataStore:    *storage.NewStorage(ttl),
		keyStore:     make(map[string]chan string),
		alpha:        ALPHA,
		beta:         BETA,
		k:            K,
		ttl:          ttl,
		// *storage.NewStorageWithTTL(60 * time.Second),
	}

	network := NewNetwork(contact, conn, kademlia.HandleMessage)

	kademlia.Network = network

	go kademlia.Network.Listen()
	go kademlia.managePendingRequests()
	go kademlia.RunPeriodicCleanup(5 * time.Second)

	return kademlia, nil
}

func (k *Kademlia) managePendingRequests() {
	pending := make(map[string]chan Message)

	for req := range k.mapManagerCh {
		if req.register {
			pending[req.rpcID.String()] = req.responseChan
		} else {
			if ch, ok := pending[req.rpcID.String()]; ok {
				if !req.responseMsg.RPCID.IsZero() {
					ch <- req.responseMsg
				}
				delete(pending, req.rpcID.String())
			}
		}
	}
}

func (kademlia *Kademlia) JoinNetwork(knownContact *Contact) {
	//1. Create ID if not exists
	if kademlia.Self.ID == nil {
		kademlia.Self.ID = NewRandomKademliaID()
	}

	//2. Insert known contact into routing table (correct bucket)
	kademlia.RoutingTable.AddContact(*knownContact)

	//3. Run an Iterative Find Node on Self
	kademlia.IterativeFindNode(kademlia.Self.ID, ALPHA, kademlia.k)

	//4. Refresh bucket further away than closest
	// neighbor
	closest := kademlia.RoutingTable.FindClosestContacts(kademlia.Self.ID, 1)
	bucketIndex := kademlia.RoutingTable.getBucketIndex(closest[0].ID)
	for i := bucketIndex + 1; i < IDLength*8; i++ {
		kademlia.RefreshBucket(i)
	}
}

func (kademlia *Kademlia) RefreshBucket(idx int) {
	contact := kademlia.RoutingTable.buckets[idx].getContactForBucketRefresh()
	if contact.ID != nil {
		kademlia.IterativeFindNode(contact.ID, ALPHA, kademlia.k)
	}
}

// Helper function to get the outbound IP address
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func (kademlia *Kademlia) RunPeriodicCleanup(interval time.Duration) {

	for {
		time.Sleep(interval)
		kademlia.DataStore.Clean()
	}
}

// PeriodicReplication implements the 1-hour replication rule.
// It iterates over all data this node holds and re-stores it on the
// k-closest nodes to ensure data persists even if nodes leave.
func (kademlia *Kademlia) PeriodicReplication(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Starting periodic replication cycle...")
		keys := kademlia.DataStore.GetKeys()
		for _, keyStr := range keys {
			value, _, found := kademlia.DataStore.GetValueAndMetadataForReplication(keyStr)
			if found {
				go kademlia.IterativeStore(value, true)
			}
		}
	}
}

// Shutdown gracefully
func (kademlia *Kademlia) Shutdown() {
	close(kademlia.mapManagerCh)
	if kademlia.httpServer != nil {
		if err := kademlia.httpServer.Close(); err != nil {
			log.Printf("Error shutting down HTTP server: %v", err)
		}
	}
}
