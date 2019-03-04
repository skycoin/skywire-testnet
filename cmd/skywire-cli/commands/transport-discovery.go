package commands

import (
	"github.com/spf13/cobra"

	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/cipher"
	trClient "github.com/skycoin/skywire/pkg/transport-discovery/client"
)

type transportDiscoveryCMD struct {
	root            *cobra.Command
	transportByID   *cobra.Command
	transportByEdge *cobra.Command

	flags struct {
		addr string
	}
}

func newTransportDiscoveryCmds() *cobra.Command {
	t := &transportDiscoveryCMD{}
	t.initRoot()
	t.initGetTransportByID()
	t.initGetTransportByEdge()

	t.root.AddCommand(t.transportByID)
	t.root.AddCommand(t.transportByEdge)

	return t.root
}

func (t *transportDiscoveryCMD) initRoot() {
	t.root = &cobra.Command{
		Use:   "transport-discovery",
		Short: "manage operations with transport discovery api",
	}

	t.root.Flags().StringVar(&t.flags.addr, "addr",
		"localhost:9091", "address of transport discovery server")
}

func (t *transportDiscoveryCMD) initGetTransportByID() {
	t.transportByID = &cobra.Command{
		Use:   "id [id]",
		Short: "return information related to the transport referred by it's ID",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			pk, sk := cipher.GenerateKeyPair()

			tdc, err := trClient.NewHTTP("http://"+t.flags.addr, pk, sk)
			catch(err)

			id, err := uuid.Parse(args[0])
			catch(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			entry, err := tdc.GetTransportByID(ctx, id)
			catch(err)

			fmt.Println(entry)
		},
	}
}

func (t *transportDiscoveryCMD) initGetTransportByEdge() {
	t.transportByEdge = &cobra.Command{
		Use:   "edge [edge-public-key]",
		Short: "return information related to the transport referred by it's edge pk",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			pk, sk := cipher.GenerateKeyPair()

			tdc, err := trClient.NewHTTP("http://"+t.flags.addr, pk, sk)
			catch(err)

			edgePK := cipher.PubKey{}
			catch(edgePK.Set(args[0]))

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			entries, err := tdc.GetTransportsByEdge(ctx, edgePK)
			catch(err)

			for _, entry := range entries {
				fmt.Println(entry)
			}
		},
	}

}
