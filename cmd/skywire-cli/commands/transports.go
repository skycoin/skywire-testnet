package commands

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"os"
	"text/tabwriter"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/node"
)

type transportsCMD struct {
	root      *cobra.Command
	types     *cobra.Command // client.TransportTypes
	list      *cobra.Command // client.Transports
	listFlags struct {
		typesFilter []string
		pksFilter   pkSlice
		logs        bool
	}

	add      *cobra.Command // client.TransportAdd
	addFlags struct {
		transportType string
		public        bool
	}

	summary *cobra.Command // client.Transport
	rm      *cobra.Command // client.RemoveTransport
}

type pkSlice []cipher.PubKey

// String implements stringer
func (p pkSlice) String() string {
	res := "pubkey list:\n"
	for _, pk := range p {
		res += fmt.Sprintf("\t%s\n", pk)
	}

	return res
}

// Set implements pflag.Value
func (p *pkSlice) Set(list string) error {
	parsedList := strings.Split(list, " ")

	for _, s := range parsedList {
		pk := cipher.PubKey{}
		if err := pk.Set(s); err != nil {
			return err
		}
		*p = append(*p, pk)
	}

	return nil
}

// Type implements pflag.Value
func (p pkSlice) Type() string {
	return "[]transport.Pubkey"
}

func newTransportsCmds() *cobra.Command {
	t := &transportsCMD{}
	t.initRoot()
	t.initTypes()
	t.initList()
	t.initAdd()
	t.initRm()
	t.initSummary()

	t.root.AddCommand(t.types)
	t.root.AddCommand(t.list)
	t.root.AddCommand(t.add)
	t.root.AddCommand(t.rm)
	t.root.AddCommand(t.summary)

	return t.root
}

func (t *transportsCMD) tabPrint(trList ...*node.TransportSummary) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
	_, err := fmt.Fprintln(w, "type\tid\tlocal\tremote")
	catch(err)

	for _, tr := range trList {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tr.Type, tr.ID, tr.Local,
			tr.Remote)
		catch(err)
	}

	catch(w.Flush())
}

func (t *transportsCMD) initRoot() {
	t.root = &cobra.Command{
		Use:   "transports",
		Short: "manages transports related operations",
	}
}

func (t *transportsCMD) initTypes() {
	t.types = &cobra.Command{
		Use:   "types",
		Short: "display available types",
		Run: func(_ *cobra.Command, _ []string) {
			types, err := client().TransportTypes()
			catch(err)

			for _, t := range types {
				fmt.Println(t)
			}
		},
	}
}

func (t *transportsCMD) initList() {
	t.list = &cobra.Command{
		Use:   "list",
		Short: "lists the available transports with optional filter flags",
		Run: func(_ *cobra.Command, _ []string) {
			transports, err := client().Transports(t.listFlags.typesFilter,
				t.listFlags.pksFilter, t.listFlags.logs)

			catch(err)

			t.tabPrint(transports...)
		},
	}

	t.bindListFlags()
}

func (t *transportsCMD) bindListFlags() {
	t.list.Flags().StringSliceVar(&t.listFlags.typesFilter, "types", []string{},
		"list of transport's type to filter by")
	t.list.Flags().Var(&t.listFlags.pksFilter, "pks",
		"list of transport's public keys filter by")
	t.list.Flags().BoolVar(&t.listFlags.logs, "v", false,
		"whether to print logs or not")
}

func (t *transportsCMD) initAdd() {
	t.add = &cobra.Command{
		Use:   "add [transport-public-key]",
		Short: "adds a new transport",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			pk := cipher.PubKey{}
			catch(pk.Set(args[0]))

			tr, err := client().AddTransport(pk,
				t.addFlags.transportType, t.addFlags.public)

			catch(err)

			t.tabPrint(tr)
		},
	}

	t.bindAddFlags()
}

func (t *transportsCMD) bindAddFlags() {
	t.add.Flags().StringVar(&t.addFlags.transportType, "type", "messaging",
		"type of the transport to add")
	t.add.Flags().BoolVar(&t.addFlags.public, "public", true,
		"whether to add the transport as public or private")
}

func (t *transportsCMD) initRm() {
	t.rm = &cobra.Command{
		Use:   "rm [id]",
		Short: "removes transport with given id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			id, err := uuid.Parse(args[0])
			catch(err)

			catch(client().RemoveTransport(id))

			fmt.Println("OK")
		},
	}
}

func (t *transportsCMD) initSummary() {
	t.summary = &cobra.Command{
		Use:   "summary [id]",
		Short: "returns summary of given transport by id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			id, err := uuid.Parse(args[0])
			catch(err)

			s, err := client().Transport(id)
			catch(err)

			t.tabPrint(s)
		},
	}
}
