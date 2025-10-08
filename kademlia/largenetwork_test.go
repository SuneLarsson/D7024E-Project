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

	// 1. Configuration

	sim := NewSimulatedNetwork(0, seed) // Start with 0 drop rate for setup.
	nodes := make([]*Kademlia, numNodes)
	// r := rand.New(rand.NewSource(seed))

	// 2. Node Creation
	for i := 0; i < numNodes; i++ {
		addr := fmt.Sprintf("node-%d", i)
		node := NewTestKademliaNode(addr, sim)
		nodes[i] = node
	}

	coreNetworkSize := numNodes / 5

	// 4a. Create a fully interconnected core network.
	// This provides a stable set of entry points for new nodes.
	for i := 0; i < coreNetworkSize; i++ {
		for j := 0; j < coreNetworkSize; j++ {
			if i != j {
				nodes[i].RoutingTable.AddContact(nodes[j].Self)
			}
		}
	}

	var wg sync.WaitGroup
	for i := coreNetworkSize; i < numNodes; i++ {
		wg.Add(1)
		go func(nodeIndex int) {
			defer wg.Done()
			newNode := nodes[nodeIndex]
			// Select an entry point from any node that has already been bootstrapped.
			entryPointIndex := nodeIndex % coreNetworkSize
			entryPointNode := nodes[entryPointIndex]

			// Add an entry point to the routing table.
			newNode.RoutingTable.AddContact(entryPointNode.Self)

			// Perform the self-lookup to discover the network.
			newNode.IterativeFindNode(newNode.Self.ID, ALPHA, K)
		}(i)
	}
	wg.Wait()

	sim.dropRate = dropRate

	return nodes, sim
}

func TestLargeNetworkLookupNoDrops(t *testing.T) {
	const numIterations = 1
	const numNodes = 1000
	const dropRate = 0.0 // Keep drops at 0 for a predictable success case.

	for i := 0; i < numIterations; i++ {
		seed := int64(i)
		testName := fmt.Sprintf("IterationWithSeed_%d", seed)

		t.Run(testName, func(t *testing.T) {
			nodes, _ := SetupLargeNetwork(t, numNodes, dropRate, seed)
			r := rand.New(rand.NewSource(seed))

			// Select two different random nodes for the test.
			// NOTE: Corrected the random logic to be `r.Intn(numNodes)` for 0-indexed slices.
			nodeAIndex := r.Intn(numNodes)
			targetNodeIndex := r.Intn(numNodes)
			for nodeAIndex == targetNodeIndex {
				targetNodeIndex = r.Intn(numNodes)
			}

			nodeA := nodes[nodeAIndex]           // The node performing the lookup.
			targetNode := nodes[targetNodeIndex] // The node we want to find.

			contacts := nodeA.IterativeFindNode(targetNode.Self.ID, ALPHA, K)

			// The first contact in the returned list should be the exact node we were looking for.
			closestContact := contacts[0]
			assert.True(t, closestContact.ID.Equals(targetNode.Self.ID), "The closest node found should be the target node")
		})
	}
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

			for i := int64(0); i < numLookups; i++ {
				wg.Add(1)
				currentJobIndex := i

				go func() {
					defer wg.Done()

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

			assert.True(t, successRate > (1.0-dropRate), "Most lookups should succeed")

		})
	}
}
