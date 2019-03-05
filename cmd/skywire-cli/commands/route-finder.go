package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
)

func makeRouteFinderCmds() *cobra.Command {
	var addr string
	var minHops, maxHops uint16

	c := &cobra.Command{
		Use:   "route-finder",
		Short: "manage operations with route finder api",
	}

	c.PersistentFlags().StringVar(&addr, "addr",
		"https://routefinder.skywire.skycoin.net", "address in which to contact route finder service")

	routes := &cobra.Command{
		Use:   "routes [public-key-node-1] [public-key-node-2]",
		Short: "returns routes between two given nodes",
		Args:  cobra.MinimumNArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			rfc := routeFinder.NewHTTP(addr)
			originNode := cipher.PubKey{}
			destinyNode := cipher.PubKey{}
			err := originNode.Set(args[0])
			catch(err)

			err = destinyNode.Set(args[1])
			catch(err)

			forward, reverse, err := rfc.PairedRoutes(originNode, destinyNode, minHops, maxHops)
			catch(err)
			fmt.Println("forward: ", forward)
			fmt.Println("reverse: ", reverse)
		},
	}

	routes.Flags().Uint16Var(&minHops, "min-hops",
		1, "min hops for the returning routeFinderRoutesCmd")
	routes.Flags().Uint16Var(&maxHops, "max-hops",
		1000, "max hops for the returning routeFinderRoutesCmd")

	c.AddCommand(routes)

	return c
}
