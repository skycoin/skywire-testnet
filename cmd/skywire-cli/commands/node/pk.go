package node

import (
	"fmt"
	"log"

	"github.com/skycoin/skywire/cmd/skywire-cli/commands"
	"github.com/spf13/cobra"
)

func init() {
	nodeCmd.AddCommand(pkCmd)
}

var pkCmd = &cobra.Command{
	Use:   "pk",
	Short: "get public key of node",
	Run: func(_ *cobra.Command, _ []string) {

		client := commands.PrpcClient()
		summary, err := client.Summary()
		if err != nil {
			log.Fatal("Failed to connect:", err)
		}

		fmt.Println(summary.PubKey)
	},
}
