package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
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

func ListenOneLine(conn net.Conn) string {
	resp, _ := Listen(conn, func(line string) bool {
		return true // stop after first line
	})
	return resp
}

func ListenUntilEnd(conn net.Conn) (string, error) {
	return Listen(conn, func(line string) bool {
		return line == "END"
	})
}

func Listen(conn net.Conn, stopCondition func(string) bool) (string, error) {
	reader := bufio.NewReader(conn)
	var sb strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return sb.String(), io.EOF
			}
			return "", err
		}

		trimmed := strings.TrimSpace(line)
		if stopCondition(trimmed) {
			break
		}

		sb.WriteString(line) // preserve original newlines
	}
	return sb.String(), nil
}
