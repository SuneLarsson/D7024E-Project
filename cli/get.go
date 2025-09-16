package cli

import (
	"d7024e/storage"
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
		conn := storage.ConnectToServer(storage.DEFAULT_SOCKET)
		defer conn.Close()
		storage.SendMessage(conn, "get"+storage.SEPARATING_STRING+args[0])
		response := storage.ListenToResponse(conn)
		fmt.Println(response)
	},
}
