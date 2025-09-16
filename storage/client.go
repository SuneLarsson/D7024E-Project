package storage

import (
	"bufio"
	"fmt"
	"net"
)

const ERR_ABSENTSERVER string = "Socket does not exist at indicated socket path"

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

func ListenToResponse(conn net.Conn) string {

	var reply string
	reader := bufio.NewReader(conn)

	reply, _ = reader.ReadString('\n')

	return reply

}
