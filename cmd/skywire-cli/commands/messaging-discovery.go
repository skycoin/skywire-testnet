package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"os"
	"text/tabwriter"

	"github.com/skycoin/skywire/pkg/cipher"
	mdClient "github.com/skycoin/skywire/pkg/messaging-discovery/client"
)

func makeMessagingDiscoveryCmds() *cobra.Command {
	var addr string

	availableServersTabPrint := func(entries []*mdClient.Entry) {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
		_, err := fmt.Fprintln(w, "version\tregistered\tpublic-key\taddress\tport\tconns")
		catch(err)

		for _, entry := range entries {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%d\n", entry.Version, entry.Timestamp, entry.Static, entry.Server.Address,
				entry.Server.Port, entry.Server.AvailableConnections)
		}
		w.Flush()
	}

	c := &cobra.Command{
		Use:   "messaging-discovery",
		Short: "manage operations with messaging discovery api",
	}

	c.PersistentFlags().StringVar(&addr, "addr",
		"https://messaging.discovery.skywire.skycoin.net", "address of messaging discovery server")

	c.AddCommand(&cobra.Command{
		Use:   "entry [node-public-key]",
		Short: "fetch entry from messaging-discovery instance",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			c := mdClient.NewHTTP(addr)
			pk := cipher.PubKey{}
			catch(pk.Set(args[0]))

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			entry, err := c.Entry(ctx, pk)
			catch(err)

			fmt.Println(entry)
		},
	})

	c.AddCommand(&cobra.Command{
		Use:   "available-servers",
		Short: "fetch available servers from messaging-discovery instance",
		Run: func(_ *cobra.Command, _ []string) {
			c := mdClient.NewHTTP(addr)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			entries, err := c.AvailableServers(ctx)
			catch(err)

			availableServersTabPrint(entries)
		},
	})

	return c
}
