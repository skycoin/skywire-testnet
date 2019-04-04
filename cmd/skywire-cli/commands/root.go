package commands

import (
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/mdisc"
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/node"
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/rtfind"
	"github.com/skycoin/skywire/cmd/skywire-cli/commands/tpdisc"
	"github.com/spf13/cobra"
)

var rpcAddr string

var rootCmd = &cobra.Command{
	Use:   "skywire-cli",
	Short: "Command Line Interface for skywire",
}

func init() {
	rootCmd.AddCommand(
		node.NodeCmd,
		mdisc.MessageDiscoveryCmd,
		rtfind.RtFindCmd,
		tpdisc.TransportCmd,
	)
}

// Execute executes root CLI command.
func Execute() {
	rootCmd.PersistentFlags().StringVarP(&rpcAddr, "rpc", "", "localhost:3435", "RPC server address")
	rootCmd.Execute() //nolint:errcheck
}
