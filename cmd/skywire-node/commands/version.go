package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/node"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the skywire-node version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(node.Version)
	},
}
