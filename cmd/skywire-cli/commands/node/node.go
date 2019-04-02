package node

import (
	"github.com/skycoin/skywire/cmd/skywire-cli/commands"
	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Commands that interact with the skywire-node",
}

func init() {
	commands.AddCommand(nodeCmd)
}
