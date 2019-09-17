package tpdisc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/spf13/cobra"

	"github.com/SkycoinProject/skywire/cmd/skywire-cli/internal"
	"github.com/SkycoinProject/skywire/pkg/transport"
	"github.com/SkycoinProject/skywire/pkg/transport-discovery/client"
)

var (
	addr string
	tpID transportID
	tpPK cipher.PubKey
)

func init() {
	RootCmd.Flags().StringVar(&addr, "addr", "https://transport.discovery.skywire.skycoin.net", "address of transport discovery")
	RootCmd.Flags().Var(&tpID, "id", "if specified, obtains a single transport of given ID")
	RootCmd.Flags().Var(&tpPK, "pk", "if specified, obtains transports associated with given public key")
}

// RootCmd is the command that queries the transport-discovery.
var RootCmd = &cobra.Command{
	Use:   "tpdisc (--id=<transport-id> | --pk=<edge-public-key>)",
	Short: "Queries the Transport Discovery to find transport(s) of given transport ID or edge public key",
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
		internal.Catch(err)
		if tpPK.Null() {
			entry, err := c.GetTransportByID(ctx, uuid.UUID(tpID))
			internal.Catch(err)
			printTransportEntries(entry)
		} else {
			entries, err := c.GetTransportsByEdge(ctx, pk)
			internal.Catch(err)
			printTransportEntries(entries...)
		}
	},
}

func printTransportEntries(entries ...*transport.EntryWithStatus) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
	_, err := fmt.Fprintln(w, "id\ttype\tpublic\tregistered\tup\tedge1\tedge2\topinion1\topinion2")
	internal.Catch(err)
	for _, e := range entries {
		_, err := fmt.Fprintf(w, "%s\t%s\t%t\t%d\t%t\t%s\t%s\t%t\t%t\n",
			e.Entry.ID, e.Entry.Type, e.Entry.Public, e.Registered, e.IsUp, e.Entry.Edges[0], e.Entry.Edges[1], e.Statuses[0], e.Statuses[1])
		internal.Catch(err)
	}
	internal.Catch(w.Flush())
}

type transportID uuid.UUID

// String implements pflag.Value
func (t transportID) String() string { return uuid.UUID(t).String() }

// Type implements pflag.Value
func (transportID) Type() string { return "transportID" }

// Set implements pflag.Value
func (t *transportID) Set(s string) error {
	tID, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	*t = transportID(tID)
	return nil
}
