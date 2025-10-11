package cli

import (
	"d7024e/server"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(routingCmd)
}

var routingCmd = &cobra.Command{
	Use:   "routing",
	Short: "Show routing table of node",
	Long:  "Show routing table of node",
	Run: func(cmd *cobra.Command, args []string) {
		response := routing()
		fmt.Println(response)
	},
}

func routing() string {
	conn := server.ConnectToServer(server.Default_socket)
	defer conn.Close()

	server.SendMessage(conn, "routing")

	response, err := server.ListenUntilEnd(conn)
	if err != nil && err != io.EOF {
		fmt.Println("Error reading response:", err)
		response = "Error reading response:" + err.Error()
	}
	return response
}
