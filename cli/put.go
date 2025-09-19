package cli

import (
	"d7024e/server"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(PutCmd)
}

var PutCmd = &cobra.Command{
	Use:   "put",
	Short: "Upload a file",
	Long:  "Upload a file",
	Run: func(cmd *cobra.Command, args []string) {
		conn := server.ConnectToServer(server.DEFAULT_SOCKET)
		defer conn.Close()
		server.SendMessage(conn, "put"+server.SEPARATING_STRING+args[0])
		response := server.ListenToResponse(conn)
		fmt.Println("Value stored at key", response)
	},
}
