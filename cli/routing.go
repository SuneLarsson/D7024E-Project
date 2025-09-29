package cli

import (
	"d7024e/server"
	"fmt"

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
		conn := server.ConnectToServer(server.DEFAULT_SOCKET)
		defer conn.Close()
		server.SendMessage(conn, "routing")
		for {
			response := server.ListenToResponse(conn)
			if response == "END" {
				break
			}
			fmt.Println(response)
		}
	},
}
