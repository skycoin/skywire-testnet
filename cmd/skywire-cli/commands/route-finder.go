package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
)

type routeFinderCmds struct {
	root        *cobra.Command
	routes      *cobra.Command
	routesFlags struct {
		minHops uint16
		maxHops uint16
		address string
	}
}

func newRouteFinderCmds() *cobra.Command {
	r := &routeFinderCmds{}
	r.initRootCMD()
	r.initRoutesCMD()
	r.initRoutesFlags()

	r.root.AddCommand(r.routes)

	return r.root
}

func (r *routeFinderCmds) initRootCMD() {
	r.root = &cobra.Command{
		Use:   "route-finder",
		Short: "manage operations with route finder api",
	}
}

func (r *routeFinderCmds) initRoutesCMD() {
	r.routes = &cobra.Command{
		Use:   "routes [public-key-node-1] [public-key-node-2]",
		Short: "returns routes between two given nodes",
		Args:  cobra.MinimumNArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			rfc := routeFinder.NewHTTP("http://" + r.routesFlags.address)
			originNode := cipher.PubKey{}
			destinyNode := cipher.PubKey{}
			err := originNode.Set(args[0])
			catch(err)

			err = destinyNode.Set(args[1])
			catch(err)

			forward, reverse, err := rfc.PairedRoutes(originNode, destinyNode,
				r.routesFlags.minHops, r.routesFlags.maxHops)
			catch(err)
			fmt.Println("forward: ", forward)
			fmt.Println("reverse: ", reverse)
		},
	}
}

func (r *routeFinderCmds) initRoutesFlags() {
	r.routes.Flags().Uint16Var(&r.routesFlags.minHops, "min-hops",
		1, "min hops for the returning routeFinderRoutesCmd")

	r.routes.Flags().Uint16Var(&r.routesFlags.maxHops, "max-hops",
		1000, "max hops for the returning routeFinderRoutesCmd")

	r.routes.Flags().StringVar(&r.routesFlags.address, "addr",
		"localhost:9092", "address in which to contact route finder service")
}
