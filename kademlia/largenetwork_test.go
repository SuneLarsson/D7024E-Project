package kademlia

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func SetupLargeNetwork(t *testing.T, numNodes int, dropRate float64, seed int64) ([]*Kademlia, *SimulatedNetwork) {

	// Phase 1: Create all node instances (sequentially, safe).
	sim := NewSimulatedNetwork(0, seed) // Start with 0 drop rate for setup.
	nodes := make([]*Kademlia, numNodes)
	for i := 0; i < numNodes; i++ {
		addr := fmt.Sprintf("node-%d", i)
		node := NewTestKademliaNode(addr, sim)
		nodes[i] = node
	}

	if numNodes == 0 {
		return nodes, sim
	}

	// Phase 2: Build a stable, interconnected core network (sequentially, safe).
	// This provides a robust set of entry points for new nodes.
	coreNetworkSize := numNodes / 10
	if coreNetworkSize == 0 {
		coreNetworkSize = 1 // Ensure at least one node in the core if the network is small.
	}

	for i := 0; i < coreNetworkSize; i++ {
		for j := 0; j < coreNetworkSize; j++ {
			if i != j {
				nodes[i].RoutingTable.AddContact(nodes[j].Self)
			}
		}
	}

	// Phase 3: Join the remaining nodes to the core network (sequentially, safe).
	// Each node fully joins before the next one starts.
	for i := coreNetworkSize; i < numNodes; i++ {
		newNode := nodes[i]
		entryPointIndex := i % coreNetworkSize

		// Select a random entry point from the stable core network.
		entryPointNode := nodes[entryPointIndex]

		// Use the proper JoinNetwork method to bootstrap the node.
		// This is a blocking call, so the network is stable before the next iteration.
		newNode.JoinNetwork(&entryPointNode.Self)
	}
	// Now that the setup is complete and stable, set the real drop rate for the test.
	sim.dropRate = dropRate

	return nodes, sim
}

func TestLargeNetworkLookupDrops(t *testing.T) {
	const numIterations = 1
	const numNodes = 1000
	const dropRate = 0.05

	for i := 0; i < numIterations; i++ {
		seed := int64(i)
		testName := fmt.Sprintf("DropIterationWithSeed_%d", seed)

		t.Run(testName, func(t *testing.T) {
			nodes, _ := SetupLargeNetwork(t, numNodes, dropRate, seed)

			numLookups := int64(numNodes)

			var wg sync.WaitGroup
			var successesCount int64 = 0
			const concurrencyLimit = 100
			sem := make(chan struct{}, concurrencyLimit)

			for i := int64(0); i < numLookups; i++ {
				wg.Add(1)
				currentJobIndex := i

				sem <- struct{}{}

				go func() {
					defer wg.Done()
					defer func() { <-sem }()

					r := rand.New(rand.NewSource(currentJobIndex))

					nodeAIndex := r.Intn(numNodes)
					targetNodeIndex := r.Intn(numNodes)
					for nodeAIndex == targetNodeIndex {
						targetNodeIndex = r.Intn(numNodes)
					}

					nodeA := nodes[nodeAIndex]
					targetNode := nodes[targetNodeIndex]

					contacts := nodeA.IterativeFindNode(targetNode.Self.ID, ALPHA, K)

					if len(contacts) == 0 {
						return
					}

					closestContact := contacts[0]
					if closestContact.ID.Equals(targetNode.Self.ID) {
						atomic.AddInt64(&successesCount, 1)
					}
				}()
			}

			wg.Wait()

			successRate := float64(successesCount) / float64(numLookups)
			fmt.Printf("Lookup success rate with %.2f drop rate: %.2f%% (%d/%d)\n", dropRate, successRate*100, successesCount, numLookups)
			log.Printf("Lookup success rate with %.2f drop rate: %.2f%% (%d/%d)\n", dropRate, successRate*100, successesCount, numLookups)

			assert.True(t, successRate > (0.9), "Most lookups should succeed")

		})
	}
}

func TestLargeNetworkLookup(t *testing.T) {
	const numIterations = 1
	const numNodes = 1000
	const dropRate = 0.00

	for i := 0; i < numIterations; i++ {
		seed := int64(i)
		testName := fmt.Sprintf("NoDropIterationWithSeed_%d", seed)

		t.Run(testName, func(t *testing.T) {
			nodes, _ := SetupLargeNetwork(t, numNodes, dropRate, seed)

			numLookups := int64(numNodes)

			var wg sync.WaitGroup
			var successesCount int64 = 0
			const concurrencyLimit = 100
			sem := make(chan struct{}, concurrencyLimit)

			for i := int64(0); i < numLookups; i++ {
				wg.Add(1)
				currentJobIndex := i

				sem <- struct{}{}

				go func() {
					defer wg.Done()
					defer func() { <-sem }()

					r := rand.New(rand.NewSource(currentJobIndex))

					nodeAIndex := r.Intn(numNodes)
					targetNodeIndex := r.Intn(numNodes)
					for nodeAIndex == targetNodeIndex {
						targetNodeIndex = r.Intn(numNodes)
					}

					nodeA := nodes[nodeAIndex]
					targetNode := nodes[targetNodeIndex]

					contacts := nodeA.IterativeFindNode(targetNode.Self.ID, ALPHA, K)

					if len(contacts) == 0 {
						return
					}

					closestContact := contacts[0]
					if closestContact.ID.Equals(targetNode.Self.ID) {
						atomic.AddInt64(&successesCount, 1)
					}
				}()
			}

			wg.Wait()

			successRate := float64(successesCount) / float64(numLookups)
			fmt.Printf("Lookup success rate with %.2f drop rate: %.2f%% (%d/%d)\n", dropRate, successRate*100, successesCount, numLookups)
			log.Printf("Lookup success rate with %.2f drop rate: %.2f%% (%d/%d)\n", dropRate, successRate*100, successesCount, numLookups)

			assert.True(t, successRate > (0.95), "Most lookups should succeed")

		})
	}
}
