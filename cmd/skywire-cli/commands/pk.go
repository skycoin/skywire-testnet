package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pkCmd)
}

var pkCmd = &cobra.Command{
	Use:   "pk",
	Short: "get public key of node",
	Run: func(_ *cobra.Command, _ []string) {

		client := rpcClient()
		summary, err := client.Summary()
		if err != nil {
			log.Fatal("Failed to connect:", err)
		}

		fmt.Println(summary.PubKey)
	},
}
