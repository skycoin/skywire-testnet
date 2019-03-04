package commands

import (
	"fmt"
	"log"
	"net/rpc"
	"os"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/manager"
	"github.com/skycoin/skywire/pkg/node"
)

var rpcAddr string

func init() {
	rootCmd.AddCommand(newAppCmds())

	rootCmd.AddCommand(newRouteFinderCmds())

	rootCmd.AddCommand(newTransportDiscoveryCmds())

	rootCmd.AddCommand(newMessagingDiscoveryCmds())

	rootCmd.AddCommand(newTransportsCmds())

	rootCmd.AddCommand(newRoutingRulesCmds())
}

func client() node.RPCClient {
	client, err := rpc.Dial("tcp", rpcAddr)
	if err != nil {
		log.Fatal("RPC connection failed:", err)
	}
	return manager.NewRPCClient(client, node.RPCPrefix)
}

func catch(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "skywire-cli",
	Short: "Command Line Interface for skywire",
}

// Execute executes root CLI command.
func Execute() {
	rootCmd.PersistentFlags().StringVarP(&rpcAddr, "rpc", "", "localhost:3436", "RPC server address")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
