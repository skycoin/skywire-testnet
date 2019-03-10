package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
)

func init() {
	rootCmd.AddCommand(messagingCmd)
}

var mdAddr string

var messagingCmd = &cobra.Command{
	Use:   "messaging",
	Short: "manage operations with messaging services",
}

func init() {
	messagingCmd.PersistentFlags().StringVar(&mdAddr, "addr", "https://messaging.discovery.skywire.skycoin.net", "address of messaging discovery server")

	messagingCmd.AddCommand(
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
		pk := parsePK("node-public-key", args[0])
		entry, err := client.NewHTTP(mdAddr).Entry(ctx, pk)
		catch(err)
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
		catch(err)
		printAvailableServers(entries)
	},
}

func printAvailableServers(entries []*client.Entry) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
	_, err := fmt.Fprintln(w, "version\tregistered\tpublic-key\taddress\tport\tconns")
	catch(err)
	for _, entry := range entries {
		_, err := fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%d\n",
			entry.Version, entry.Timestamp, entry.Static, entry.Server.Address, entry.Server.Port, entry.Server.AvailableConnections)
		catch(err)
	}
	catch(w.Flush())
}
