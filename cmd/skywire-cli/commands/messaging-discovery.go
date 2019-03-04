package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/cipher"
	mdClient "github.com/skycoin/skywire/pkg/messaging-discovery/client"
)

type messagingDiscoveryCmds struct {
	flags struct {
		addr string
	}

	root                *cobra.Command
	entry               *cobra.Command
	getAvailableServers *cobra.Command
}

func newMessagingDiscoveryCmds() *cobra.Command {
	m := &messagingDiscoveryCmds{}
	m.initRoot()
	m.initEntry()
	m.initGetAvailableServers()

	m.root.AddCommand(m.entry)
	m.root.AddCommand(m.getAvailableServers)

	return m.root
}

func (m *messagingDiscoveryCmds) initRoot() {
	m.root = &cobra.Command{
		Use:   "messaging-discovery",
		Short: "manage operations with messaging discovery api",
	}

	m.root.Flags().StringVar(&m.flags.addr, "addr",
		"localhost:9090", "address of messaging discovery server")
}

func (m *messagingDiscoveryCmds) initEntry() {
	m.entry = &cobra.Command{
		Use:   "entry [node-public-key]",
		Short: "fetch entry from messaging-discovery instance",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			c := mdClient.NewHTTP("http://" + m.flags.addr)
			pk := cipher.PubKey{}
			catch(pk.Set(args[0]))

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			entry, err := c.Entry(ctx, pk)
			catch(err)

			fmt.Println(entry)
		},
	}
}

func (m *messagingDiscoveryCmds) initGetAvailableServers() {
	m.getAvailableServers = &cobra.Command{
		Use:   "available-servers",
		Short: "fetch available servers from messaging-discovery instance",
		Run: func(_ *cobra.Command, _ []string) {
			c := mdClient.NewHTTP("http://" + m.flags.addr)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			entries, err := c.AvailableServers(ctx)
			catch(err)

			for _, entry := range entries {
				fmt.Println(entry)
			}
		},
	}
}
