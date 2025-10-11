package cli

import (
	"d7024e/server"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(getCmd)
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a value",
	Long:  "Get a value",
	Run: func(cmd *cobra.Command, args []string) {
		response := get(args[0])
		fmt.Println(response)
	},
}

func get(arg string) string {
	conn := server.ConnectToServer(server.Default_socket)
	defer conn.Close()
	server.SendMessage(conn, "get"+server.SEPARATING_STRING+arg)
	response := server.ListenOneLine(conn)
	return response
}
