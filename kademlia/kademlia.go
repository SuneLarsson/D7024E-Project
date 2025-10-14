package kademlia

import (
	"d7024e/storage"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// const kSize = 20 // Bucket size
// const alpha = 3  // Concurrency

// Kademlia represents a node in the Kademlia DHT. It encapsulates the
// network transport, routing table, local storage, lookup parameters, and
// lifecycle management (background maintenance and shutdown).
//
// A Kademlia instance starts background goroutines for network listening,
// pending-request bookkeeping, storage cleanup, and periodic replication.
// Use Shutdown to stop these goroutines gracefully.
//
// Most fields are internal implementation details and not intended for
// direct manipulation by callers.
//
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
	httpMutex    sync.Mutex
	alpha        int
	beta         int
	k            int
	ttl          time.Duration
	done         chan struct{}
	wg           sync.WaitGroup
}

// MapRequest is an internal control message used by managePendingRequests to
// register and resolve pending RPC requests by their RPCID. Callers should not
// construct or send these directly; use higher-level RPC mechanisms instead.
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

// NewKademliaNode creates and initializes a new Kademlia node.
//
// The node binds a UDP socket on 0.0.0.0:port for inbound traffic and derives
// the public-facing address from the outbound interface (used for contact info).
// It initializes routing, local storage (using TTL), network transport, and
// starts background goroutines for listening, request management, periodic
// cleanup, and replication.
//
// Returns the initialized node or an error if socket setup fails.
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

	ttl := time.Duration(TTL) * time.Second
	log.Println("TTL: ", ttl)

	kademlia := &Kademlia{
		Self:         contact,
		mapManagerCh: make(chan MapRequest),
		DataStore:    *storage.NewStorage(ttl, int64(tExpire)),
		keyStore:     make(map[string]chan string),
		alpha:        ALPHA,
		beta:         BETA,
		k:            K,
		ttl:          ttl,
		done:         make(chan struct{}, 4),
		// *storage.NewStorageWithTTL(60 * time.Second),
	}

	routingtable := NewRoutingTable(kademlia)

	kademlia.RoutingTable = routingtable

	network := NewNetwork(contact, conn, kademlia.HandleMessage)

	kademlia.Network = network

	// Start background goroutines for network I/O, request tracking,
	// storage maintenance, and key replication. WaitGroup is used to
	// coordinate shutdown of maintenance routines.
	kademlia.wg.Add(4)
	go kademlia.Network.Listen()
	go kademlia.managePendingRequests()
	go kademlia.RunPeriodicCleanup(5 * time.Second)
	go kademlia.PeriodicReplication(time.Duration(tReplicate) * time.Hour)
	go kademlia.startRefreshLoop()

	return kademlia, nil
}

// managePendingRequests tracks pending RPC requests keyed by RPCID and routes
// incoming responses to the correct waiting channel. It terminates when a
// shutdown signal is received via kademlia.done.
func (k *Kademlia) managePendingRequests() {
	defer k.wg.Done()
	pending := make(map[string]chan Message)

	for {
		select {
		case req, ok := <-k.mapManagerCh:
			if !ok {
				// Channel closed, exit the goroutine
				return
			}
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
		case <-k.done:
			// Received shutdown signal, exit the goroutine
			return
		}

		// for req := range k.mapManagerCh {
		// 	if req.register {
		// 		pending[req.rpcID.String()] = req.responseChan
		// 	} else {
		// 		if ch, ok := pending[req.rpcID.String()]; ok {
		// 			if !req.responseMsg.RPCID.IsZero() {
		// 				ch <- req.responseMsg
		// 			}
		// 			delete(pending, req.rpcID.String())
		// 		}
		// 	}

		// }
	}
}

// JoinNetwork connects this node to the overlay using a known bootstrap
// contact. It ensures the node has an ID, inserts the bootstrap contact in the
// appropriate bucket, runs an IterativeFindNode on itself to populate nearby
// buckets, then refreshes buckets farther than the closest neighbor.
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
	// closest := kademlia.RoutingTable.FindClosestContacts(kademlia.Self.ID, 1)
	kademlia.RoutingTable.bucketsMutex.Lock()
	limit := len(kademlia.RoutingTable.buckets)
	kademlia.RoutingTable.bucketsMutex.Unlock()
	// bucketIndex := kademlia.RoutingTable.getBucketIndex(closest[0].ID)
	log.Println(kademlia.RoutingTable.String())
	for i := 0; i < limit; i++ {
		kademlia.RefreshBucket(i)
		log.Println("Refreshed bucket", i)
	}
	log.Println(kademlia.RoutingTable.String())
}

// RefreshBucket triggers a bucket-refresh operation for the bucket at index i.
// It generates a random ID within the bucket's range and initiates an
// iterative find node process.
func (kademlia *Kademlia) RefreshBucket(idx int) {
	kademlia.RoutingTable.bucketsMutex.RLock()
	// Ensure the index is valid
	if idx < 0 || idx >= len(kademlia.RoutingTable.buckets) {
		log.Println("Invalid bucket index:", idx)
		kademlia.RoutingTable.bucketsMutex.RUnlock()
		return
	}
	randomID := kademlia.RoutingTable.buckets[idx].generateRandomIDForRefresh()
	kademlia.RoutingTable.bucketsMutex.RUnlock()

	if randomID != nil {
		log.Println("Refreshing bucket", idx, "with random ID", randomID.String())
		// Perform the lookup using the new random ID
		kademlia.IterativeFindNode(randomID, ALPHA, kademlia.k)
	}
}

// getOutboundIP determines the local machine's primary outbound IP address by
// initiating a UDP connection to a well-known address and reading the local
// socket address chosen by the OS.
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// RunPeriodicCleanup periodically invokes DataStore.Clean at the given
// interval to purge expired entries. It exits when a shutdown signal is
// received via kademlia.done.
func (kademlia *Kademlia) RunPeriodicCleanup(interval time.Duration) {
	defer kademlia.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			kademlia.DataStore.Clean()
		case <-kademlia.done:
			return
		}
	}

}

// PeriodicReplication performs periodic replication of locally stored data.
// At each interval, it iterates over local keys and attempts to re-store their
// values on the k-closest nodes, improving resilience against churn. The
// interval is provided by the caller (typically derived from configuration).
func (kademlia *Kademlia) PeriodicReplication(interval time.Duration) {
	defer kademlia.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Starting periodic replication cycle...")
			keys := kademlia.DataStore.GetKeys()
			for _, keyStr := range keys {
				value, _, found := kademlia.DataStore.GetValueAndMetadataForReplication(keyStr)
				if found {
					go kademlia.IterativeStore(value, true)
				}
			}
		case <-kademlia.done:
			// Shutdown signal received, exit.
			return
		}
	}

}

// Shutdown gracefully stops background goroutines, network listeners, and the
// optional HTTP server, waiting for maintenance routines to terminate before
// returning.
func (kademlia *Kademlia) Shutdown() {

	for i := 0; i < 4; i++ {
		kademlia.done <- struct{}{}
	}
	defer close(kademlia.done)
	kademlia.wg.Wait()

	if network, ok := kademlia.Network.(*Network); ok {
		network.Shutdown()
	}

	close(kademlia.mapManagerCh)

	kademlia.httpMutex.Lock()
	if kademlia.httpServer != nil {
		if err := kademlia.httpServer.Close(); err != nil {
			log.Printf("Error shutting down HTTP server: %v", err)
		}
	}
	kademlia.httpMutex.Unlock()
}

// NewKademliaIDFromBinary creates a KademliaID from a binary string representation.
func NewKademliaIDFromBinary(binString string) (*KademliaID, error) {
	if len(binString) != IDLength*8 {
		return nil, fmt.Errorf("binary string must be %d chars long, but was %d", IDLength*8, len(binString))
	}

	var id KademliaID
	for i := 0; i < IDLength; i++ {
		// Get an 8-bit chunk (a byte)
		byteString := binString[i*8 : (i+1)*8]

		// Parse it from base 2
		byteVal, err := strconv.ParseUint(byteString, 2, 8)
		if err != nil {
			return nil, err
		}
		id[i] = uint8(byteVal)
	}
	return &id, nil
}

func (kademlia *Kademlia) startRefreshLoop() {
	defer kademlia.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Perform an initial refresh shortly after startup.
	kademlia.refreshAllBuckets()

	for {
		select {
		case <-ticker.C:
			// This case is triggered every 10 minutes.
			log.Println("Starting periodic bucket refresh...")
			kademlia.refreshAllBuckets()

		case <-kademlia.done:
			// If a value is sent on the quit channel, exit the loop.
			log.Println("Stopping bucket refresh loop.")
			return
		}
	}
}

// Helper function to contain the refresh logic.
func (kademlia *Kademlia) refreshAllBuckets() {
	kademlia.RoutingTable.bucketsMutex.RLock()
	limit := len(kademlia.RoutingTable.buckets)
	kademlia.RoutingTable.bucketsMutex.RUnlock()

	log.Printf("Refreshing %d buckets...\n", limit)
	for i := 0; i < limit; i++ {
		// Calling the function we created in the last step
		kademlia.RefreshBucket(i)
	}
	log.Println("Finished refreshing buckets.")
	log.Println(kademlia.RoutingTable.String())
}
