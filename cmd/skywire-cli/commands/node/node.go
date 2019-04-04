package node

import (
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/spf13/cobra"
)

var log = logging.MustGetLogger("skywire-cli")

//NodeCmd contains commands that interact with the skywire-node
var NodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Commands that interact with the skywire-node",
}
