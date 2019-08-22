package disc

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/skycoin/dmsg/disc"
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/cmd/skywire-cli/internal"
)

var mdAddr string

func init() {
	RootCmd.PersistentFlags().StringVar(&mdAddr, "addr", "https://dmsg.discovery.skywire.skycoin.net", "address of dmsg discovery server")
}

// RootCmd is the command that contains sub-commands which interacts with dmsg services.
var RootCmd = &cobra.Command{
	Use:   "disc",
	Short: "Contains sub-commands that interact with a remote dmsg discovery",
}

func init() {
	RootCmd.AddCommand(
		entryCmd,
		availableServersCmd,
	)
}

var entryCmd = &cobra.Command{
	Use:   "entry <node-public-key>",
	Short: "fetches an entry from dmsg-discovery",
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
	Short: "fetch available servers from dmsg-discovery",
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
