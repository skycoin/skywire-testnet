package mdisc

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/cmd/skywire-cli/internal"

	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
)

// MessageDiscoveryCmd contains commands that interact with messaging services
var MessageDiscoveryCmd = &cobra.Command{
	Use:   "mdisc",
	Short: "Commands that interact with messaging-discovery",
}

var mdAddr string

func init() {
	MessageDiscoveryCmd.PersistentFlags().StringVar(&mdAddr, "addr", "https://messaging.discovery.skywire.skycoin.net", "address of messaging discovery server")

	MessageDiscoveryCmd.AddCommand(
		mEntryCmd,
		mAvailableServersCmd)
}

var mEntryCmd = &cobra.Command{
	Use:   "entry <node-public-key>",
	Short: "fetch entry from messaging-discovery",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		pk := internal.ParsePK("node-public-key", args[0])
		entry, err := client.NewHTTP(mdAddr).Entry(ctx, pk)
		internal.Catch(err)
		fmt.Println(entry)
	},
}

var mAvailableServersCmd = &cobra.Command{
	Use:   "available-servers",
	Short: "fetch available servers from messaging-discovery",
	Run: func(_ *cobra.Command, _ []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		entries, err := client.NewHTTP(mdAddr).AvailableServers(ctx)
		internal.Catch(err)
		printAvailableServers(entries)
	},
}

func printAvailableServers(entries []*client.Entry) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
	_, err := fmt.Fprintln(w, "version\tregistered\tpublic-key\taddress\tport\tconns")
	internal.Catch(err)
	for _, entry := range entries {
		_, err := fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%d\n",
			entry.Version, entry.Timestamp, entry.Static, entry.Server.Address, entry.Server.Port, entry.Server.AvailableConnections)
		internal.Catch(err)
	}
	internal.Catch(w.Flush())
}
