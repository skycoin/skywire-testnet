package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"time"

	"github.com/google/uuid"

	"io"
	"os"
	"text/tabwriter"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/node"
	"github.com/skycoin/skywire/pkg/router"
	"github.com/skycoin/skywire/pkg/routing"
)

func makeRulesCmds() *cobra.Command {
	createWriter := func() *tabwriter.Writer {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
		_, err := fmt.Fprintln(w, "id\ttype\tlocal-port\tremote-port\tremote-pk\tresp-id\tnext-route-id\tnext-transport-id\texpire-at")
		catch(err)

		return w
	}
	tabPrintAppRule := func(w io.Writer, id routing.RouteID, s *routing.RuleSummary) {
		_, err := fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%s\t%d\t%s\t%s\t%s\n", id, s.Type, s.AppFields.LocalPort,
			s.AppFields.RemotePort, s.AppFields.RemotePK, s.AppFields.RespRID, "-", "-", s.ExpireAt)
		catch(err)
	}

	tabPrintForwardRule := func(w io.Writer, id routing.RouteID, s *routing.RuleSummary) {
		_, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n", id, s.Type, "-",
			"-", "-", "-", s.ForwardFields.NextRID, s.ForwardFields.NextTID, s.ExpireAt)
		catch(err)
	}

	tabPrint := func(rules ...*node.RoutingEntry) {
		w := createWriter()

		for _, rule := range rules {
			if rule.Value.Summary().AppFields != nil {
				tabPrintAppRule(w, rule.Key, rule.Value.Summary())
			} else {
				tabPrintForwardRule(w, rule.Key, rule.Value.Summary())
			}
		}
		catch(w.Flush())
	}

	tabPrintSingleRule := func(id routing.RouteID, s *routing.RuleSummary) {
		w := createWriter()
		if s.AppFields != nil {
			tabPrintAppRule(w, id, s)
		} else {
			tabPrintForwardRule(w, id, s)
		}

		w.Flush()
	}

	c := &cobra.Command{
		Use:   "routing-rules [sub-command]",
		Short: "manages operations with routing rules",
	}
	c.AddCommand(makeAddCmds())

	c.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "print the list of current routing rules",
		Run: func(_ *cobra.Command, _ []string) {
			rules, err := client().RoutingRules()
			catch(err)

			tabPrint(rules...)
		},
	})

	c.AddCommand(&cobra.Command{
		Use:   "info [routing-rule-id]",
		Short: "returns information about the routing-rule referenced by it's id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			id, err := strconv.ParseUint(args[0], 10, 32)
			catch(err)

			rule, err := client().RoutingRule(routing.RouteID(id))
			catch(err)

			tabPrintSingleRule(rule.RouteID(), rule.Summary())
		},
	})

	c.AddCommand(&cobra.Command{
		Use:   "rm [routing-rule-id]",
		Short: "removes given routing-rule by it's id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			id, err := strconv.ParseUint(args[0], 10, 32)
			catch(err)

			catch(client().RemoveRoutingRule(routing.RouteID(id)))

			fmt.Println("OK")
		},
	})

	return c
}

func makeAddCmds() *cobra.Command {
	var expireAfter time.Duration
	var localPort, remotePort uint16

	c := &cobra.Command{
		Use:   "add [app|forward]",
		Short: "adds a new rule",
	}

	c.PersistentFlags().DurationVar(&expireAfter, "expire", router.RouteTTL,
		"duration after which rule will expire")

	app := &cobra.Command{
		Use:   "app [route-id] [remote-pk]",
		Short: "adds an app rule",
		Args:  cobra.MinimumNArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			id, err := strconv.ParseUint(args[0], 10, 32)
			catch(err)

			pk := cipher.PubKey{}
			catch(pk.Set(args[1]))

			rule := routing.AppRule(time.Now().Add(expireAfter),
				routing.RouteID(id), pk, remotePort, localPort)

			resID, err := client().AddRoutingRule(rule)
			catch(err)

			fmt.Println("id: ", resID)
		},
	}
	app.Flags().Uint16Var(&localPort, "local-port", 80,
		"local port of the rule")
	app.Flags().Uint16Var(&remotePort, "remote-port", 80,
		"remote port of the rule")

	c.AddCommand(app)

	c.AddCommand(&cobra.Command{
		Use:   "forward [next-route-id] [next-transport-id]",
		Short: "adds a forward rule",
		Args:  cobra.MinimumNArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			routeID, err := strconv.ParseUint(args[0], 10, 32)
			catch(err)

			nTrID, err := uuid.Parse(args[1])
			catch(err)

			rule := routing.ForwardRule(time.Now().Add(expireAfter),
				routing.RouteID(routeID), nTrID)

			id, err := client().AddRoutingRule(rule)
			catch(err)

			fmt.Println("id: ", id)
		},
	})

	return c
}
