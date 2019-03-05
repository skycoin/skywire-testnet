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

func makeTransportsCmds() *cobra.Command {
	var (
		typesFilter []string
		pksFilter   pkSlice
		logs        bool
	)

	tabPrint := func(trList ...*node.TransportSummary) {
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
	c := &cobra.Command{
		Use:   "transports",
		Short: "manages transports related operations",
	}

	c.AddCommand(&cobra.Command{
		Use:   "types",
		Short: "display available types",
		Run: func(_ *cobra.Command, _ []string) {
			types, err := client().TransportTypes()
			catch(err)

			for _, t := range types {
				fmt.Println(t)
			}
		},
	})

	list := &cobra.Command{
		Use:   "list",
		Short: "lists the available transports with optional filter flags",
		Run: func(_ *cobra.Command, _ []string) {
			transports, err := client().Transports(typesFilter,
				pksFilter, logs)

			catch(err)

			tabPrint(transports...)
		},
	}
	list.Flags().StringSliceVar(&typesFilter, "types", []string{},
		"list of transport's type to filter by")
	list.Flags().Var(&pksFilter, "pks",
		"list of transport's public keys filter by")
	list.Flags().BoolVar(&logs, "v", false,
		"whether to print logs or not")
	c.AddCommand(list)

	c.AddCommand(&cobra.Command{
		Use:   "rm [id]",
		Short: "removes transport with given id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			id, err := uuid.Parse(args[0])
			catch(err)

			catch(client().RemoveTransport(id))

			fmt.Println("OK")
		},
	})

	c.AddCommand(&cobra.Command{
		Use:   "summary [id]",
		Short: "returns summary of given transport by id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			id, err := uuid.Parse(args[0])
			catch(err)

			s, err := client().Transport(id)
			catch(err)

			tabPrint(s)
		},
	})

	return c
}
