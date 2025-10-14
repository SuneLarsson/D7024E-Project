package cli

import (
	"d7024e/server"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(putCmd)
}

var putCmd = &cobra.Command{
	Use:   "put",
	Short: "Upload a file",
	Long:  "Upload a file",
	Run: func(cmd *cobra.Command, args []string) {
		value := strings.Join(args, " ")
		response := put(value)
		fmt.Fprintln(output, "Value stored at key", response)
	},
}

func put(arg string) string {
	conn := server.ConnectToServer(server.Default_socket)
	defer conn.Close()
	server.SendMessage(conn, "put"+server.SEPARATING_STRING+arg)
	response := server.ListenOneLine(conn)
	return response
}
