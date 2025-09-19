package server

import (
	"os"
	"testing"
)

func TestNoServer(t *testing.T) {
	defer func() {
		if err := recover(); err == nil || err != ERR_ABSENTSERVER {
			t.Error("When no server is running, ERR_ABSENTSERVER should be thrown")
		}
	}()

	os.Remove(DEFAULT_SOCKET)
	ConnectToServer(DEFAULT_SOCKET)
}
