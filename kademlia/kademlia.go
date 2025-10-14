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
	httpMutex    sync.Mutex
	alpha        int
	beta         int
	k            int
	ttl          time.Duration
	done         chan struct{}
	wg           sync.WaitGroup
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

	ttl := time.Duration(TTL) * time.Second

	kademlia := &Kademlia{
		Self:         contact,
		mapManagerCh: make(chan MapRequest),
		DataStore:    *storage.NewStorage(ttl, int64(tExpire)),
		keyStore:     make(map[string]chan string),
		alpha:        ALPHA,
		beta:         BETA,
		k:            K,
		ttl:          ttl,
		done:         make(chan struct{}, 3),
		// *storage.NewStorageWithTTL(60 * time.Second),
	}

	routingtable := NewRoutingTable(kademlia)

	kademlia.RoutingTable = routingtable

	network := NewNetwork(contact, conn, kademlia.HandleMessage)

	kademlia.Network = network

	kademlia.wg.Add(3)
	go kademlia.Network.Listen()
	go kademlia.managePendingRequests()
	go kademlia.RunPeriodicCleanup(5 * time.Second)
	go kademlia.PeriodicReplication(time.Duration(tReplicate) * time.Hour)

	return kademlia, nil
}

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
	kademlia.RoutingTable.bucketsMutex.Lock()
	limit := len(kademlia.RoutingTable.buckets)
	kademlia.RoutingTable.bucketsMutex.Unlock()
	bucketIndex := kademlia.RoutingTable.getBucketIndex(closest[0].ID)
	for i := bucketIndex + 1; i < limit; i++ {
		kademlia.RefreshBucket(i)
	}
}

func (kademlia *Kademlia) RefreshBucket(idx int) {
	kademlia.RoutingTable.bucketsMutex.Lock()
	contact := kademlia.RoutingTable.buckets[idx].getContactForBucketRefresh()
	kademlia.RoutingTable.bucketsMutex.Unlock()
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
	// for {
	// 	time.Sleep(interval)
	// 	kademlia.DataStore.Clean()
	// }
}

// PeriodicReplication implements the 1-hour replication rule.
// It iterates over all data this node holds and re-stores it on the
// k-closest nodes to ensure data persists even if nodes leave.
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

	// for range ticker.C {
	// 	log.Println("Starting periodic replication cycle...")
	// 	keys := kademlia.DataStore.GetKeys()
	// 	for _, keyStr := range keys {
	// 		value, _, found := kademlia.DataStore.GetValueAndMetadataForReplication(keyStr)
	// 		if found {
	// 			go kademlia.IterativeStore(value, true)
	// 		}
	// 	}
	// }
}

// Shutdown gracefully
func (kademlia *Kademlia) Shutdown() {

	for i := 0; i < 3; i++ {
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
