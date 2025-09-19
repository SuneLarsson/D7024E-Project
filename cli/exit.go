package cli

import (
	"d7024e/server"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(ExitCmd)
}

var ExitCmd = &cobra.Command{
	Use:   "exit",
	Short: "Terminate the node",
	Long:  "Terminate the node",
	Run: func(cmd *cobra.Command, args []string) {
		conn := server.ConnectToServer(server.DEFAULT_SOCKET)
		defer conn.Close()
		server.SendMessage(conn, "exit")
	},
}
