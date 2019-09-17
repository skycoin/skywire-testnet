package rtfind

import (
	"fmt"
	"time"

	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/spf13/cobra"

	"github.com/SkycoinProject/skywire-mainnet/cmd/skywire-cli/internal"
	"github.com/SkycoinProject/skywire-mainnet/pkg/route-finder/client"
)

var frAddr string
var frMinHops, frMaxHops uint16
var timeout time.Duration

func init() {
	RootCmd.Flags().StringVar(&frAddr, "addr", "https://routefinder.skywire.skycoin.net", "address in which to contact route finder service")
	RootCmd.Flags().Uint16Var(&frMinHops, "min-hops", 1, "min hops for the returning routeFinderRoutesCmd")
	RootCmd.Flags().Uint16Var(&frMaxHops, "max-hops", 1000, "max hops for the returning routeFinderRoutesCmd")
	RootCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "timeout for remote server requests")
}

// RootCmd is the command that queries the route-finder.
var RootCmd = &cobra.Command{
	Use:   "rtfind <public-key-node-1> <public-key-node-2>",
	Short: "Queries the Route Finder for available routes between two nodes",
	Args:  cobra.MinimumNArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		rfc := client.NewHTTP(frAddr, timeout)

		var srcPK, dstPK cipher.PubKey
		internal.Catch(srcPK.Set(args[0]))
		internal.Catch(dstPK.Set(args[1]))

		forward, reverse, err := rfc.PairedRoutes(srcPK, dstPK, frMinHops, frMaxHops)
		internal.Catch(err)

		fmt.Println("forward: ", forward)
		fmt.Println("reverse: ", reverse)
	},
}
