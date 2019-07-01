package commands

import (
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/cmd/skywire-cli/commands/mdisc"
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/rtfind"
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/tpdisc"
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/visor"
)

var rootCmd = &cobra.Command{
	Use:   "skywire-cli",
	Short: "Command Line Interface for skywire",
}

func init() {
	rootCmd.AddCommand(
		visor.RootCmd,
		mdisc.RootCmd,
		rtfind.RootCmd,
		tpdisc.RootCmd,
	)
}

// Execute executes root CLI command.
func Execute() {
	_ = rootCmd.Execute() //nolint:errcheck
}
