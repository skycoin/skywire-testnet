package node

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/cmd/skywire-cli/internal"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/node"
)

func init() {
	RootCmd.AddCommand(
		transportTypesCmd,
		listTransportsCmd,
		transportCmd,
		addTransportCmd,
		rmTransportCmd,
	)
}

var transportTypesCmd = &cobra.Command{
	Use:   "transport-types",
	Short: "lists transport types used by the local node",
	Run: func(_ *cobra.Command, _ []string) {
		types, err := rpcClient().TransportTypes()
		internal.Catch(err)
		for _, t := range types {
			fmt.Println(t)
		}
	},
}

var (
	filterTypes   []string
	filterPubKeys cipher.PubKeys
	showLogs      bool
)

var listTransportsCmd = &cobra.Command{
	Use:   "list-transports",
	Short: "lists the available transports with optional filter flags",
	Run: func(_ *cobra.Command, _ []string) {
		transports, err := rpcClient().Transports(filterTypes, filterPubKeys, showLogs)
		internal.Catch(err)
		printTransports(transports...)
	},
}

func init() {
	listTransportsCmd.Flags().StringSliceVar(&filterTypes, "filter-types", filterTypes, "comma-separated; if specified, only shows transports of given types")
	listTransportsCmd.Flags().Var(&filterPubKeys, "filter-pks", "comma-separated; if specified, only shows transports associated with given nodes")
	listTransportsCmd.Flags().BoolVar(&showLogs, "show-logs", true, "whether to show transport logs in output")
}

var transportCmd = &cobra.Command{
	Use:   "transport <transport-id>",
	Short: "returns summary of given transport by id",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		tpID := internal.ParseUUID("transport-id", args[0])
		tp, err := rpcClient().Transport(tpID)
		internal.Catch(err)
		printTransports(tp)
	},
}

var (
	transportType string
	public        bool
	timeout       time.Duration
)

var addTransportCmd = &cobra.Command{
	Use:   "add-transport <remote-public-key>",
	Short: "adds a new transport",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		pk := internal.ParsePK("remote-public-key", args[0])
		tp, err := rpcClient().AddTransport(pk, transportType, public, timeout)
		internal.Catch(err)
		printTransports(tp)
	},
}

func init() {
	addTransportCmd.Flags().StringVar(&transportType, "type", "messaging", "type of transport to add")
	addTransportCmd.Flags().BoolVar(&public, "public", true, "whether to make the transport public")
	addTransportCmd.Flags().DurationVarP(&timeout, "timeout", "t", 0, "if specified, sets an operation timeout")
}

var rmTransportCmd = &cobra.Command{
	Use:   "rm-transport <transport-id>",
	Short: "removes transport with given id",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		tID := internal.ParseUUID("transport-id", args[0])
		internal.Catch(rpcClient().RemoveTransport(tID))
		fmt.Println("OK")
	},
}

func printTransports(tps ...*node.TransportSummary) {
	sortTransports(tps...)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
	_, err := fmt.Fprintln(w, "type\tid\tlocal\tremote")
	internal.Catch(err)
	for _, tp := range tps {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tp.Type, tp.ID, tp.Local, tp.Remote)
		internal.Catch(err)
	}
	internal.Catch(w.Flush())
}

func sortTransports(tps ...*node.TransportSummary) {
	sort.Slice(tps, func(i, j int) bool {
		return tps[i].ID.String() < tps[j].ID.String()
	})
}
