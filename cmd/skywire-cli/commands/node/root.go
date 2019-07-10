package node

import (
	"net/rpc"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/visor"
)

var log = logging.MustGetLogger("skywire-cli")

var rpcAddr string

func init() {
	RootCmd.PersistentFlags().StringVarP(&rpcAddr, "rpc", "", "localhost:3435", "RPC server address")
}

// RootCmd contains commands that interact with the skywire-networking-node
var RootCmd = &cobra.Command{
	Use:   "node",
	Short: "Contains sub-commands that interact with the local Skywire Networking Node",
}

func rpcClient() visor.RPCClient {
	client, err := rpc.Dial("tcp", rpcAddr)
	if err != nil {
		log.Fatal("RPC connection failed:", err)
	}
	return visor.NewRPCClient(client, visor.RPCPrefix)
}
