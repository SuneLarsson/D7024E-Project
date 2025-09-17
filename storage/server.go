package storage

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const SEPARATING_STRING string = ":"
const DEFAULT_SOCKET string = "/tmp/svc.sock"

type Server struct {
	socketPath string
	exitNode   bool
	mutExit    sync.RWMutex
	storage    *Storage
}

func NewServer(sockPath string) *Server {
	return &Server{socketPath: sockPath, exitNode: false}
}

// Starts begin listening for incoming messages
func (s Server) Listen() {
	os.Remove(s.socketPath)

	s.storage = NewStorage()

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	connCh := make(chan net.Conn)
	errCh := make(chan error)

	for {
		s.mutExit.RLock()
		if s.exitNode {
			s.mutExit.RUnlock()
			break
		} else {
			s.mutExit.RUnlock()
		}

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- err
				return
			}
			connCh <- conn
		}()

		select {
		case conn := <-connCh:
			go s.handleConnection(conn)
		case err := <-errCh:
			//TODO
			fmt.Println("Error on connection:", err)
		case <-time.After(1 * time.Second):

		}

	}
	ln.Close()
	os.Remove(s.socketPath)
}

// Handle the connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewScanner(conn)

	for reader.Scan() {
		request := strings.TrimSpace(reader.Text())

		splitRequest := strings.Split(request, SEPARATING_STRING)

		if len(splitRequest) < 1 {
			panic("Message shoud at least contain type of message")
		}

		switch splitRequest[0] {
		case "exit":
			s.mutExit.Lock()
			s.exitNode = true
			s.mutExit.Unlock()
		case "ping":
			reply(conn, "pong")
		case "get":
			var response string
			response, _ = s.storage.Get(splitRequest[1])
			reply(conn, response)
		case "put":
			s.storage.Put(splitRequest[1], splitRequest[2])
		}
	}

	if err := reader.Err(); err != nil {
		fmt.Println("Connection closed with error:", err)
	} else {
		fmt.Println("Client disconnected.")
	}
}

// Sends a reply
func reply(conn net.Conn, reply string) {
	fmt.Fprintln(conn, reply)
}
