package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReply(t *testing.T) {

	socketPath := filepath.Join(os.TempDir(), "test-svc.sock")

	t.Cleanup(func() {
		os.Remove(socketPath)
	})

	server := NewServer(socketPath, "", 8000)

	ch := make(chan string, 1)

	go func() {
		server.Listen()
	}()

	time.Sleep(100 * time.Millisecond)

	conn := ConnectToServer(socketPath)

	go func() {
		ch <- ListenToResponse(conn)
	}()

	SendMessage(conn, "ping")

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fail()
	}
}

func TestExitWorking(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "test-svc.sock")

	t.Cleanup(func() {
		os.Remove(socketPath)
	})
	server := NewServer(socketPath, "", 8001)

	ch := make(chan string, 1)

	go func() {
		server.Listen()
		ch <- "exit"
	}()

	time.Sleep(3000 * time.Millisecond)

	conn := ConnectToServer(socketPath)

	SendMessage(conn, "exit")

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.FailNow()
	}

}
