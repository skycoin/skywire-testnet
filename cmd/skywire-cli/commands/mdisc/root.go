package mdisc

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/SkycoinProject/dmsg/disc"
	"github.com/spf13/cobra"

	"github.com/SkycoinProject/skywire-mainnet/cmd/skywire-cli/internal"
)

var mdAddr string

func init() {
	RootCmd.PersistentFlags().StringVar(&mdAddr, "addr", "https://messaging.discovery.skywire.skycoin.net", "address of messaging discovery server")
}

// RootCmd is the command that contains sub-commands which interacts with messaging services.
var RootCmd = &cobra.Command{
	Use:   "mdisc",
	Short: "Contains sub-commands that interact with a remote Messaging Discovery",
}

func init() {
	RootCmd.AddCommand(
		entryCmd,
		availableServersCmd,
	)
}

var entryCmd = &cobra.Command{
	Use:   "entry <node-public-key>",
	Short: "fetches an entry from messaging-discovery",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		pk := internal.ParsePK("node-public-key", args[0])
		entry, err := disc.NewHTTP(mdAddr).Entry(ctx, pk)
		internal.Catch(err)
		fmt.Println(entry)
	},
}

var availableServersCmd = &cobra.Command{
	Use:   "available-servers",
	Short: "fetch available servers from messaging-discovery",
	Run: func(_ *cobra.Command, _ []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		entries, err := disc.NewHTTP(mdAddr).AvailableServers(ctx)
		internal.Catch(err)
		printAvailableServers(entries)
	},
}

func printAvailableServers(entries []*disc.Entry) {
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
