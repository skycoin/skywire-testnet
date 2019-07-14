package commands

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/cmd/skywire-cli/commands/mdisc"
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/node"
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/rtfind"
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/tpdisc"
)

var rootCmd = &cobra.Command{
	Use:   "skywire-cli",
	Short: "Command Line Interface for skywire",
}

func init() {
	rootCmd.AddCommand(
		node.RootCmd,
		mdisc.RootCmd,
		rtfind.RootCmd,
		tpdisc.RootCmd,
	)
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal("Failed to execute command: ", err)
	}
}
