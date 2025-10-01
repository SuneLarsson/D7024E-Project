package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

const ERR_ABSENTSERVER string = "Socket does not exist at indicated socket path"

type ResponseReader struct {
	Reader *bufio.Reader
}

func NewResponseReader(conn net.Conn) *ResponseReader {
	return &ResponseReader{
		Reader: bufio.NewReader(conn),
	}
}

func ConnectToServer(socketPath string) net.Conn {

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		panic(ERR_ABSENTSERVER)
	}

	return conn
}

func SendMessage(conn net.Conn, messageType string) {

	SendMessageWithArgument(conn, messageType, "")

}

func SendMessageWithArgument(conn net.Conn, messageType string, argument string) {

	toSend := messageType + SEPARATING_STRING + argument
	fmt.Fprintln(conn, toSend)

}

func (rr *ResponseReader) ListenToResponse() string {

	// var reply string

	reply, err := rr.Reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			log.Print("Connection closed by server")
			return "END"
		}
		return "END" // fail safe
	}
	fmt.Println("DEBUG: Raw reply from server:", reply)
	// return reply //without trailing newline
	return strings.TrimSpace(reply)

}
