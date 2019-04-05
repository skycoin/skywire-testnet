package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/node"
	"github.com/skycoin/skywire/pkg/transport"
	"github.com/skycoin/skywire/pkg/transport-discovery/client"
)

func init() {
	rootCmd.AddCommand(
		transportTypesCmd,
		listTransportsCmd,
		transportCmd,
		addTransportCmd,
		rmTransportCmd,
		findTransport)
}

var transportTypesCmd = &cobra.Command{
	Use:   "transport-types",
	Short: "lists transport types used by the local node",
	Run: func(_ *cobra.Command, _ []string) {
		types, err := rpcClient().TransportTypes()
		catch(err)
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
		catch(err)
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
		tpID := parseUUID("transport-id", args[0])
		tp, err := rpcClient().Transport(tpID)
		catch(err)
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
		pk := parsePK("remote-public-key", args[0])
		tp, err := rpcClient().AddTransport(pk, transportType, public, timeout)
		catch(err)
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
		tID := parseUUID("transport-id", args[0])
		catch(rpcClient().RemoveTransport(tID))
		fmt.Println("OK")
	},
}

var (
	addr string
	tpID transportID
	tpPK cipher.PubKey
)

var findTransport = &cobra.Command{
	Use:   "find-transport (--id=<transport-id> | --pk=<edge-public-key>)",
	Short: "finds and lists transport(s) of given transport ID or edge public key from transport discovery",
	Args: func(_ *cobra.Command, _ []string) error {
		var (
			nilID = uuid.UUID(tpID) == (uuid.UUID{})
			nilPK = tpPK.Null()
		)
		if nilID && nilPK {
			return errors.New("must specify --id flag or --pk flag")
		}
		if !nilID && !nilPK {
			return errors.New("cannot specify --id and --pk flag")
		}
		return nil
	},
	Run: func(_ *cobra.Command, _ []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pk, sk := cipher.GenerateKeyPair()
		c, err := client.NewHTTP(addr, pk, sk)
		catch(err)
		if tpPK.Null() {
			entry, err := c.GetTransportByID(ctx, uuid.UUID(tpID))
			catch(err)
			printTransportEntries(entry)
		} else {
			entries, err := c.GetTransportsByEdge(ctx, pk)
			catch(err)
			printTransportEntries(entries...)
		}
	},
}

func init() {
	findTransport.Flags().StringVar(&addr, "addr", "https://transport.discovery.skywire.skycoin.net", "address of transport discovery")
	findTransport.Flags().Var(&tpID, "id", "if specified, obtains a single transport of given ID")
	findTransport.Flags().Var(&tpPK, "pk", "if specified, obtains transports associated with given public key")
}

func printTransports(tps ...*node.TransportSummary) {
	sortTransports(tps...)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
	_, err := fmt.Fprintln(w, "type\tid\tlocal\tremote")
	catch(err)
	for _, tp := range tps {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tp.Type, tp.ID, tp.Local, tp.Remote)
		catch(err)
	}
	catch(w.Flush())
}

func printTransportEntries(entries ...*transport.EntryWithStatus) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
	_, err := fmt.Fprintln(w, "id\ttype\tpublic\tregistered\tup\tedge1\tedge2\topinion1\topinion2")
	catch(err)
	for _, e := range entries {
		_, err := fmt.Fprintf(w, "%s\t%s\t%t\t%d\t%t\t%s\t%s\t%t\t%t\n",
			e.Entry.ID, e.Entry.Type, e.Entry.Public, e.Registered, e.IsUp, e.Entry.Edges()[0], e.Entry.Edges()[1], e.Statuses[0], e.Statuses[1])
		catch(err)
	}
	catch(w.Flush())
}

func sortTransports(tps ...*node.TransportSummary) {
	sort.Slice(tps, func(i, j int) bool {
		return tps[i].ID.String() < tps[j].ID.String()
	})
}
