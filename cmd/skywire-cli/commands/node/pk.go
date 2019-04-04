package node

import (
	"fmt"

	"github.com/skycoin/skywire/cmd/skywire-cli/internal"

	"github.com/spf13/cobra"
)

func init() {
	NodeCmd.AddCommand(pkCmd)
}

var pkCmd = &cobra.Command{
	Use:   "pk",
	Short: "get public key of node",
	Run: func(_ *cobra.Command, _ []string) {

		client := internal.RPCClient()
		summary, err := client.Summary()
		if err != nil {
			log.Fatal("Failed to connect:", err)
		}

		fmt.Println(summary.PubKey)
	},
}
