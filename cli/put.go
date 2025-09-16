package cli

import (
	"d7024e/storage"

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
		conn := storage.ConnectToServer(storage.DEFAULT_SOCKET)
		defer conn.Close()
		storage.SendMessage(conn, "put"+storage.SEPARATING_STRING+args[0]+storage.SEPARATING_STRING+args[1])
	},
}
