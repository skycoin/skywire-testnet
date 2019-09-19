package node

import (
	"net/rpc"

	"github.com/SkycoinProject/skycoin/src/util/logging"
	"github.com/spf13/cobra"

	"github.com/SkycoinProject/skywire-mainnet/pkg/visor"
)

var log = logging.MustGetLogger("skywire-cli")

var rpcAddr string

func init() {
	RootCmd.PersistentFlags().StringVarP(&rpcAddr, "rpc", "", "localhost:3435", "RPC server address")
}

// RootCmd contains commands that interact with the skywire-visor
var RootCmd = &cobra.Command{
	Use:   "node",
	Short: "Contains sub-commands that interact with the local Skywire Visor",
}

func rpcClient() visor.RPCClient {
	client, err := rpc.Dial("tcp", rpcAddr)
	if err != nil {
		log.Fatal("RPC connection failed:", err)
	}
	return visor.NewRPCClient(client, visor.RPCPrefix)
}
