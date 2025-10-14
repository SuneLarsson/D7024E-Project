package kademlia

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

/*func TestShowRoutingTable(t *testing.T) {

	for i := 1; i <= 20; i++ {
		go printRoutingTable(i)
	}
	time.Sleep(1 * time.Minute)
	t.Error("Test")
}*/

func printRoutingTable(BETA int) {
	sim := NewSimulatedNetwork(0, 0)
	watchingNode := NewTestKademliaNodeWithID(NewKademliaID("0000000000000000000000000000000000000000"), "watching", sim)

	rt := watchingNode.RoutingTable
	rt.buckets[0].b = BETA

	for i := 0; i < 1000; i++ {
		node := NewTestKademliaNode("node"+strconv.Itoa(i), sim)
		rt.AddContact(node.Self)
		time.Sleep(50 * time.Millisecond)
	}

	rt.bucketsMutex.Lock()

	tree := buildTree(rt.buckets)

	rt.bucketsMutex.Unlock()

	exportToDot(tree, BETA)

}

type TreeNode struct {
	Prefix string
	Length int
	Left   *TreeNode
	Right  *TreeNode
}

func buildTree(buckets []*bucket) *TreeNode {
	var root *TreeNode
	for _, b := range buckets {
		length := 0
		if b.list != nil {
			length = b.list.Len()
		}
		root = insertNode(root, b.prefix, length)
	}
	return root
}

func insertNode(root *TreeNode, prefix string, length int) *TreeNode {
	if root == nil {
		root = &TreeNode{}
	}

	if prefix == "" {
		// Leaf node
		root.Prefix = prefix
		root.Length = length
		return root
	}

	current := root
	for i, ch := range prefix {
		if ch == '0' {
			if current.Left == nil {
				current.Left = &TreeNode{}
			}
			if i == len(prefix)-1 {
				current.Left.Prefix = prefix
				current.Left.Length = length
			}
			current = current.Left
		} else if ch == '1' {
			if current.Right == nil {
				current.Right = &TreeNode{}
			}
			if i == len(prefix)-1 {
				current.Right.Prefix = prefix
				current.Right.Length = length
			}
			current = current.Right
		}
	}
	return root
}

func exportToDot(root *TreeNode, betaValue int) error {
	_, err := os.Stat("images")
	if err != nil {
		os.Mkdir("images", 0755)
	}
	filename := "images/" + strconv.Itoa(betaValue)
	f, err := os.Create(filename + ".dot")
	if err != nil {
		return err
	}
	defer f.Close()

	// Start DOT graph definition
	fmt.Fprintf(f, "digraph G {\n")
	fmt.Fprintf(f, "label=\"BETA = %d\";\nlabelloc=top;\nfontsize=20;\n", betaValue)
	fmt.Fprintf(f, "node [shape=circle, style=filled, fillcolor=lightgrey];\n")

	// Recursive function to write nodes and edges
	var writeEdges func(node *TreeNode, id string)
	writeEdges = func(node *TreeNode, id string) {
		if node == nil {
			return
		}

		label := node.Prefix
		if node.Prefix != "" {
			label = fmt.Sprintf("%s (%d)", node.Prefix, node.Length)
		}
		fmt.Fprintf(f, "%s [label=\"%s\"];\n", id, label)

		if node.Left != nil {
			leftID := fmt.Sprintf("%s0", id)
			fmt.Fprintf(f, "%s -> %s [label=\"0\"];\n", id, leftID)
			writeEdges(node.Left, leftID)
		}
		if node.Right != nil {
			rightID := fmt.Sprintf("%s1", id)
			fmt.Fprintf(f, "%s -> %s [label=\"1\"];\n", id, rightID)
			writeEdges(node.Right, rightID)
		}
	}

	writeEdges(root, "root")
	fmt.Fprintf(f, "}\n")
	f.Close()

	// Generate PNG (requires Graphviz installed)
	cmd := exec.Command("dot", "-Tpng", filename+".dot", "-o", filename+".png")
	return cmd.Run()
}
