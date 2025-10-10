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
		conn := server.ConnectToServer(server.DEFAULT_SOCKET)
		defer conn.Close()

		server.SendMessage(conn, "routing")

		response, err := server.ListenUntilEnd(conn)
		if err != nil && err != io.EOF {
			fmt.Println("Error reading response:", err)
			return
		}

		fmt.Print(response)
		// rr := server.NewResponseReader(conn)
		// for {
		// 	response, err := rr.ListenToResponse(conn)
		// 	if err != nil {
		// 		fmt.Println("Error reading response:", err)
		// 		break
		// 	}
		// 	fmt.Print(response)
		// }
	},
}
