package server

import (
	"bytes"
	"d7024e/kademlia"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReply(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))

	t.Cleanup(func() {
		os.Remove(socketPath)
	})

	server := NewServer(socketPath, "", 8000)
	server.restPort = 8200

	ch := make(chan string, 1)

	go func() {
		server.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	conn := ConnectToServer(socketPath)

	go func() {
		ch <- ListenOneLine(conn)
	}()

	SendMessage(conn, "ping")

	SendMessage(conn, "exit")

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fail()
	}
}

func TestExitWorking(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))

	t.Cleanup(func() {
		os.Remove(socketPath)
	})
	server := NewServer(socketPath, "", 8001)
	server.restPort = 8201

	ch := make(chan string, 1)

	go func() {
		server.Listen()
		ch <- "exit"
	}()

	time.Sleep(300 * time.Millisecond)

	conn := ConnectToServer(socketPath)

	SendMessage(conn, "exit")

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.FailNow()
	}

}

func TestErrorSocket(t *testing.T) {

	server := NewServer("/does/not/exist", "", 8002)
	server.restPort = 8202
	ch := make(chan string, 1)
	go func() {
		defer func() {
			if err := recover(); err == nil || err != ERR_INVALIDSOCKET {
				ch <- "error"
			} else {
				ch <- "success"
			}
		}()
		server.Listen()
	}()

	select {
	case res := <-ch:
		if res == "error" {
			t.Error("When socket path is invalid, ERR_INVALIDSOCKET should be thrown")
		}
	case <-time.After(2 * time.Second):
		SendMessage(ConnectToServer("/does/not/exist"), "exit")
		t.Fail()
	}
}

func TestCreationTwoNodes(t *testing.T) {

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))
	socketPath2 := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc2-%d.sock", time.Now().UnixNano()))

	defer func() {
		if err := recover(); err == nil || err != ERR_NODECREATIONFAILURE {
			t.Error("When kademlia node values are already in use, ERR_NODECREATIONFAILURE should be thrown")
		}
		SendMessage(ConnectToServer(socketPath), "exit")
	}()

	server1 := NewServer(socketPath, "", 8003)
	server1.restPort = 8203

	go func() {
		server1.Listen()
	}()
	time.Sleep(100 * time.Millisecond)
	server2 := NewServer(socketPath2, "", 8003)
	server2.restPort = 8203
	server2.Listen()

}

func TestViaBootstrapNode(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))
	socketPath2 := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc2-%d.sock", time.Now().UnixNano()))
	ch := make(chan string, 1)
	server1 := NewServer(socketPath, "", 8004)
	server1.restPort = 8204
	go func() {
		defer func() {
			if err := recover(); err != nil {
				ch <- "error"
			}
		}()
		server1.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	server2 := NewServer(socketPath2, "127.0.0.1:8004", 8005)
	server2.restPort = 8205
	go func() {
		defer func() {
			if err := recover(); err != nil {
				ch <- "error"
			}
		}()
		server2.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	select {
	case <-ch:
		t.Error("No error should be thrown when bootstrapping with a valid node")
	case <-time.After(2 * time.Second):
		SendMessage(ConnectToServer(socketPath), "exit")
		SendMessage(ConnectToServer(socketPath2), "exit")
	}

}

func TestNonExistingBootstrap(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))

	ch := make(chan string, 1)
	server := NewServer(socketPath, "127.0.0.1:8100", 8006)
	server.restPort = 8206
	go func() {
		defer func() {
			if err := recover(); err == nil || err != ERR_NODECREATIONFAILURE {
				ch <- "error"
			}
			ch <- "ok"
		}()
		server.Listen()
	}()
	select {
	case res := <-ch:
		if res == "error" {
			t.Error("When bootstrap node does not exist, ERR_NODECREATIONFAILURE should be thrown")
		}
	case <-time.After(90 * time.Second):
		SendMessage(ConnectToServer(socketPath), "exit")
		t.Fail()
	}
}

func TestForget(t *testing.T) {
	os.Setenv("TTL", "3")
	os.Setenv("tRepublish", "3")

	kademlia.ReloadConfig()

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))
	socketPath2 := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc2-%d.sock", time.Now().UnixNano()))

	fmt.Println("Initialising first server")
	server := NewServer(socketPath, "", 8007)
	server.restPort = 8207

	go func() {
		server.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	fmt.Println("Initialising second server")

	server2 := NewServer(socketPath2, "127.0.0.1:8007", 8008)
	server2.restPort = 8208

	go func() {
		server2.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	conn := ConnectToServer(socketPath2)

	SendMessage(conn, "put"+SEPARATING_STRING+"Hello World")
	key := ListenOneLine(conn)

	time.Sleep(100 * time.Millisecond)

	SendMessage(conn, "forget"+SEPARATING_STRING+key)

	time.Sleep(100 * time.Millisecond)

	ListenOneLine(conn)

	time.Sleep(6 * time.Second)

	SendMessage(conn, "get"+SEPARATING_STRING+key)
	response := ListenOneLine(conn)
	slices := strings.Split(response, "")
	fmt.Println(slices)

	if response != "Value not found" {
		t.Error("After forgetting a value and waiting for TTL to expire, the value should not be found")
	}

	SendMessage(ConnectToServer(socketPath), "exit")
	SendMessage(ConnectToServer(socketPath2), "exit")

	os.Unsetenv("TTL")
	os.Unsetenv("tRepublish")

	kademlia.ReloadConfig()

}

func TestInvalidKey(t *testing.T) {

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))

	server := NewServer(socketPath, "", 8009)
	server.restPort = 8209

	go func() {
		server.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	conn := ConnectToServer(socketPath)

	SendMessage(conn, "get"+SEPARATING_STRING+"invalidkey")

	response := ListenOneLine(conn)

	if response != "Invalid key" {
		t.Error("When getting a value with an invalid key, the response should be 'Invalid key'")
	}

	SendMessage(ConnectToServer(socketPath), "exit")

}

func TestPutGet(t *testing.T) {

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))

	server := NewServer(socketPath, "", 8010)
	server.restPort = 8210

	go func() {
		server.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	socketPath2 := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc2-%d.sock", time.Now().UnixNano()))

	server2 := NewServer(socketPath2, "127.0.0.1:8010", 8011)
	server2.restPort = 8211

	go func() {
		server2.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	conn := ConnectToServer(socketPath2)

	SendMessage(conn, "put"+SEPARATING_STRING+"Hello World")
	key := ListenOneLine(conn)

	SendMessage(conn, "get"+SEPARATING_STRING+key)

	response := ListenOneLine(conn)

	if response != "Hello World" {
		t.Error("When putting and getting a value, the response should be the original value")
	}

	SendMessage(ConnectToServer(socketPath), "exit")
	SendMessage(ConnectToServer(socketPath2), "exit")

}

func TestInvalidPut(t *testing.T) {

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))

	server := NewServer(socketPath, "", 8012)
	server.restPort = 8212

	go func() {
		server.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	conn := ConnectToServer(socketPath)

	SendMessage(conn, "put"+SEPARATING_STRING+"")

	response := ListenOneLine(conn)

	if response != "Value not stored" {
		t.Error("When putting an invalid value, the response should be 'Value not stored'")
	}

	SendMessage(ConnectToServer(socketPath), "exit")

}

func TestRouting(t *testing.T) {

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))

	server := NewServer(socketPath, "", 8013)
	server.restPort = 8213

	go func() {
		server.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	socketPath2 := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc2-%d.sock", time.Now().UnixNano()))

	server2 := NewServer(socketPath2, "127.0.0.1:8013", 8014)
	server2.restPort = 8214

	go func() {
		server2.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	conn := ConnectToServer(socketPath2)

	SendMessage(conn, "routing")

	response, err := ListenUntilEnd(conn)

	fmt.Println("---------------------------------")
	fmt.Println("Routing table:", response)

	if err != nil {
		t.Error("There should be no error when getting the routing table")
	}

	if strings.Count(response, "contact") != 1 {
		t.Error("The response should contain one contact")
	}

	if strings.Count(response, "Bucket") != 1 {
		t.Error("The response should contain one bucket")
	}

	SendMessage(ConnectToServer(socketPath), "exit")
	SendMessage(ConnectToServer(socketPath2), "exit")

}

func TestStore(t *testing.T) {
	
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc-%d.sock", time.Now().UnixNano()))

	server := NewServer(socketPath, "", 8015)
	server.restPort = 8215

	go func() {
		server.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	socketPath2 := filepath.Join(os.TempDir(), fmt.Sprintf("test-svc2-%d.sock", time.Now().UnixNano()))

	server2 := NewServer(socketPath2, "127.0.0.1:8015", 8016)
	server2.restPort = 8216

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	go func() {
		server2.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	conn := ConnectToServer(socketPath)

	SendMessage(conn, "store")

	time.Sleep(200 * time.Millisecond)

	// Capture the output
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	fmt.Println("Routing table:", output)

	if strings.Count(output, "contact") != 1 {
		t.Error("The response should contain one contact")
	}

	if strings.Count(output, "Bucket") != 1 {
		t.Error("The response should contain one bucket")
	}

	SendMessage(ConnectToServer(socketPath), "exit")
	SendMessage(ConnectToServer(socketPath2), "exit")

}
