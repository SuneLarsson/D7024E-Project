package cli

import (
	"d7024e/storage"

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
		conn := storage.ConnectToServer(storage.DEFAULT_SOCKET)
		defer conn.Close()
		storage.SendMessage(conn, "exit")
	},
}
