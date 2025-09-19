package cli

import (
	"d7024e/server"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(GetCmd)
}

var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a value",
	Long:  "Get a value",
	Run: func(cmd *cobra.Command, args []string) {
		conn := server.ConnectToServer(server.DEFAULT_SOCKET)
		defer conn.Close()
		server.SendMessage(conn, "get"+server.SEPARATING_STRING+args[0])
		response := server.ListenToResponse(conn)
		fmt.Println(response)
	},
}
