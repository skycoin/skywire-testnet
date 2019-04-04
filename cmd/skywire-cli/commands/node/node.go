package node

import (
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/spf13/cobra"

	"net/rpc"

	"github.com/skycoin/skywire/pkg/node"
)

var log = logging.MustGetLogger("skywire-cli")

// RootCmd contains commands that interact with the skywire-node
var RootCmd = &cobra.Command{
	Use:   "node",
	Short: "Commands that interact with the skywire-node",
}

var rpcAddr string

func init() {
	RootCmd.PersistentFlags().StringVarP(&rpcAddr, "rpc", "", "localhost:3435", "RPC server address")
}

func rpcClient() node.RPCClient {
	client, err := rpc.Dial("tcp", rpcAddr)
	if err != nil {
		log.Fatal("RPC connection failed:", err)
	}
	return node.NewRPCClient(client, node.RPCPrefix)
}
