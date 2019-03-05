package commands

import (
	"github.com/spf13/cobra"

	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"os"
	"text/tabwriter"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
	trClient "github.com/skycoin/skywire/pkg/transport-discovery/client"
)

func makeTransportDiscoveryCmds() *cobra.Command {
	var addr string

	tabPrint := func(entries ...*transport.EntryWithStatus) {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
		_, err := fmt.Fprintln(w, "id\ttype\tpublic\tregistered\tup\tedge1\tedge2\topinion1\topinion2")
		catch(err)

		for _, e := range entries {
			_, err := fmt.Fprintf(w, "%s\t%s\t%t\t%d\t%t\t%s\t%s\t%t\t%t\n", e.Entry.ID, e.Entry.Type, e.Entry.Public, e.Registered,
				e.IsUp, e.Entry.Edges[0], e.Entry.Edges[1], e.Statuses[0], e.Statuses[1])
			catch(err)
		}

		w.Flush()
	}

	c := &cobra.Command{
		Use:   "transport-discovery",
		Short: "manage operations with transport discovery api",
	}

	c.PersistentFlags().StringVar(&addr, "addr",
		"https://transport.discovery.skywire.skycoin.net", "address of transport discovery server")

	c.AddCommand(&cobra.Command{
		Use:   "id [id]",
		Short: "return information related to the transport referred by it's ID",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			pk, sk := cipher.GenerateKeyPair()

			tdc, err := trClient.NewHTTP(addr, pk, sk)
			catch(err)

			id, err := uuid.Parse(args[0])
			catch(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			entry, err := tdc.GetTransportByID(ctx, id)
			catch(err)

			tabPrint(entry)
		},
	})

	c.AddCommand(&cobra.Command{
		Use:   "edge [edge-public-key]",
		Short: "return information related to the transport referred by it's edge pk",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			pk, sk := cipher.GenerateKeyPair()

			tdc, err := trClient.NewHTTP(addr, pk, sk)
			catch(err)

			edgePK := cipher.PubKey{}
			catch(edgePK.Set(args[0]))

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			entries, err := tdc.GetTransportsByEdge(ctx, edgePK)
			catch(err)

			tabPrint(entries...)
		},
	})

	return c
}
