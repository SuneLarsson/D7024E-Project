package cli

import (
	"d7024e/server"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGoodBehaviour(t *testing.T) {

	myServer := server.NewServer(server.DEFAULT_SOCKET, "", 8000)
	exitCh := make(chan string, 1)
	go func() {
		myServer.Listen()
		exitCh <- "leaving"
	}()

	time.Sleep(100 * time.Millisecond)

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc2-%d.sock", time.Now().UnixNano()))
	server2 := server.NewServer(socketPath, "0.0.0.0:8000", 8001)
	server2.SetRestPort(8100)
	go func() {
		server2.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	response := put("Hello World")
	key := response

	if response == "Value not stored" {
		t.Error("Value was not stored")
	}

	response = get(key)

	if response != "Hello World" {
		fmt.Println(response)
		t.Error("The value gotten is not the same as the value put")
	}

	response = forget(key)

	if response != "This node will stop to refresh value assigned as "+key {
		t.Error("Error in message received when forgetting")
	}

	response = routing()
	fmt.Println(response)
	if strings.Count(response, "contact") != 2 {
		t.Error("There should be only two contacts in my routing table, myServer and server2")
	}

	if strings.Count(response, "Bucket") != 1 {
		t.Error("There should be only one bucket in my routing table")
	}

	exit()

	select {
	case <-exitCh:
	case <-time.After(2 * time.Second):
		t.Error("TimeOut of exiting the server")
	}

	server.SendMessage(server.ConnectToServer(socketPath), "exit")

}
