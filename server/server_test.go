package server

import (
	"testing"
	"time"
)

func TestReply(t *testing.T) {

	socketPath := DEFAULT_SOCKET
	server := NewServer(socketPath)

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
	SendMessage(conn, "exit")
}

func TestExitWorking(t *testing.T) {
	socketPath := DEFAULT_SOCKET
	server := NewServer(socketPath)

	ch := make(chan string, 1)

	go func() {
		server.Listen()
		ch <- "exit"
	}()

	time.Sleep(100 * time.Millisecond)

	conn := ConnectToServer(socketPath)

	SendMessage(conn, "exit")

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.FailNow()
	}

}
