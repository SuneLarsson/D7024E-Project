package cli

import (
	"d7024e/server"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(forgetCmd)
}

var forgetCmd = &cobra.Command{
	Use:   "forget",
	Short: "Forget a value",
	Long:  "Forget a value",
	Run: func(cmd *cobra.Command, args []string) {
		conn := server.ConnectToServer(server.DEFAULT_SOCKET)
		defer conn.Close()
		server.SendMessage(conn, "forget"+server.SEPARATING_STRING+args[0])
		response := server.ListenToResponse(conn)
		fmt.Println(response)
	},
}
