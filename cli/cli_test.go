package cli

import (
	"bytes"
	"d7024e/server"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGoodBehaviour(t *testing.T) {
	os.Setenv("ALPHA", "3")
	os.Setenv("ALPHA", "20")
	os.Setenv("ALPHA", "20")
	os.Setenv("ALPHA", "3600")
	os.Setenv("ALPHA", "3600")
	os.Setenv("ALPHA", "86400")
	os.Setenv("ALPHA", "3600")
	myServer := server.NewServer(server.Default_socket, "", 8000)
	exitCh := make(chan string, 1)
	go func() {
		myServer.Listen()
		exitCh <- "leaving"
	}()

	time.Sleep(200 * time.Millisecond)

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc2-%d.sock", time.Now().UnixNano()))
	server2 := server.NewServer(socketPath, "127.0.0.1:8000", 8001)
	server2.SetRestPort(8100)
	go func() {
		server2.Listen()
	}()

	time.Sleep(200 * time.Millisecond)

	old := os.Stdout

	// Create a pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	var output string

	// Testing the put command
	os.Args = []string{"kademlia", "put", "Hello", "World"}

	rootCmd.Execute()
	w.Close()

	buf.ReadFrom(r)
	r.Close()
	output = buf.String()
	if strings.Contains(output, "Value not stored") {
		fmt.Println(output)
		t.Error("Value was not stored")
	}

	// Getting key from the put
	key := ""
	if output != "" {
		splitOutput := strings.Split(output, " ")
		for i := len(splitOutput) - 1; i >= 0; i++ {
			if splitOutput[i] != "" {
				key = splitOutput[i]
				break
			}
		}
	}
	r, w, _ = os.Pipe()
	os.Stdout = w

	time.Sleep(100 * time.Millisecond)

	// Testing the get command
	os.Args = []string{"kademlia", "get", key}

	rootCmd.Execute()
	w.Close()

	buf.ReadFrom(r)
	r.Close()
	output = buf.String()
	if !strings.Contains(output, "Hello World") {
		//fmt.Fprintln(old, "Output:"+output+"\nEndOutput")
		t.Error("The value gotten is not the same as the value put")
	}

	time.Sleep(100 * time.Millisecond)

	r, w, _ = os.Pipe()
	os.Stdout = w

	// Testing the forget command
	os.Args = []string{"kademlia", "forget", key}

	rootCmd.Execute()
	w.Close()
	buf.ReadFrom(r)
	r.Close()
	output = buf.String()
	if !strings.Contains(output, "This node will stop to refresh value assigned as "+key) {
		t.Error("Error in message received when forgetting")
	}

	r, w, _ = os.Pipe()
	os.Stdout = w

	// Testing the routing command
	os.Args = []string{"kademlia", "routing"}

	rootCmd.Execute()
	w.Close()
	buf.ReadFrom(r)
	r.Close()
	output = buf.String()
	if strings.Count(output, "contact") != 2 {
		t.Error("There should be only two contacts in my routing table, myServer and server2")
	}

	if strings.Count(output, "Bucket") != 1 {
		t.Error("There should be only one bucket in my routing table")
	}

	// Testing the exit command
	os.Args = []string{"kademlia", "exit"}
	rootCmd.Execute()

	select {
	case <-exitCh:
	case <-time.After(2 * time.Second):
		t.Error("TimeOut of exiting the server")
	}

	// Restore stdout
	os.Stdout = old

	server.SendMessage(server.ConnectToServer(socketPath), "exit")

}
