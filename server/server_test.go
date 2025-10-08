package server

import (
	"fmt"
	"os"
	"path/filepath"
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

	server2 := NewServer(socketPath2, "0.0.0.0:8004", 8005)
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
	server := NewServer(socketPath, "0.0.0.0:8100", 8006)
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
