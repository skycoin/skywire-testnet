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
	"github.com/skycoin/skywire/pkg/routing"
)

type routingRulesCmds struct {
	root *cobra.Command
	list *cobra.Command
	info *cobra.Command
	rm   *cobra.Command
}

// addCmds refers to "add [app|forward]" sub-command
type addCmds struct {
	root     *cobra.Command
	app      *cobra.Command
	appFlags struct {
		expireAfter time.Duration
		remotePort  uint16
		localPort   uint16
	}

	forward      *cobra.Command
	forwardFlags struct {
		expireAfter time.Duration
	}
}

func newRoutingRulesCmds() *cobra.Command {
	r := &routingRulesCmds{}
	r.initRoot()
	r.initList()
	r.initInfo()
	r.initRm()

	r.root.AddCommand(r.list)
	r.root.AddCommand(r.info)
	r.root.AddCommand(r.rm)
	r.root.AddCommand(newAddCmds())

	return r.root
}

func (r *routingRulesCmds) tabPrint(rules ...*node.RoutingEntry) {
	w := r.createWriter()

	for _, rule := range rules {
		if rule.Value.Summary().AppFields != nil {
			r.tabPrintAppRule(w, rule.Key, rule.Value.Summary())
		} else {
			r.tabPrintForwardRule(w, rule.Key, rule.Value.Summary())
		}
	}

	catch(w.Flush())
}

func (r *routingRulesCmds) tabPrintSingleRule(id routing.RouteID, s *routing.RuleSummary) {
	w := r.createWriter()
	if s.AppFields != nil {
		r.tabPrintAppRule(w, id, s)
	} else {
		r.tabPrintForwardRule(w, id, s)
	}

	w.Flush()
}

func (r *routingRulesCmds) createWriter() *tabwriter.Writer {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
	_, err := fmt.Fprintln(w, "id\ttype\tlocal-port\tremote-port\tremote-pk\tresp-id\tnext-route-id\tnext-transport-id\texpire-at")
	catch(err)

	return w
}

func (r *routingRulesCmds) tabPrintAppRule(w io.Writer, id routing.RouteID, s *routing.RuleSummary) {
	_, err := fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%s\t%d\t%s\t%s\t%s\n", id, s.Type, s.AppFields.LocalPort,
		s.AppFields.RemotePort, s.AppFields.RemotePK, s.AppFields.RespRID, "-", "-", s.ExpireAt)
	catch(err)
}

func (r *routingRulesCmds) tabPrintForwardRule(w io.Writer, id routing.RouteID, s *routing.RuleSummary) {
	_, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n", id, s.Type, "-",
		"-", "-", "-", s.ForwardFields.NextRID, s.ForwardFields.NextTID, s.ExpireAt)
	catch(err)
}

func (r *routingRulesCmds) initRoot() {
	r.root = &cobra.Command{
		Use:   "routing-rules [sub-command]",
		Short: "manages operations with routing rules",
	}
}

func (r *routingRulesCmds) initList() {
	r.list = &cobra.Command{
		Use:   "list",
		Short: "print the list of current routing rules",
		Run: func(_ *cobra.Command, _ []string) {
			rules, err := client().RoutingRules()
			catch(err)

			r.tabPrint(rules...)
		},
	}
}

func (r *routingRulesCmds) initInfo() {
	r.info = &cobra.Command{
		Use:   "info [routing-rule-id]",
		Short: "returns information about the routing-rule referenced by it's id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			id, err := strconv.ParseUint(args[0], 10, 32)
			catch(err)

			rule, err := client().RoutingRule(routing.RouteID(id))
			catch(err)

			r.tabPrintSingleRule(rule.RouteID(), rule.Summary())
		},
	}
}

func (r *routingRulesCmds) initRm() {
	r.rm = &cobra.Command{
		Use:   "rm [routing-rule-id]",
		Short: "removes given routing-rule by it's id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			id, err := strconv.ParseUint(args[0], 10, 32)
			catch(err)

			catch(client().RemoveRoutingRule(routing.RouteID(id)))

			fmt.Println("OK")
		},
	}
}

func newAddCmds() *cobra.Command {
	a := &addCmds{}
	a.addRoot()
	a.addApp()
	a.addForward()

	a.root.AddCommand(a.app)
	a.root.AddCommand(a.forward)

	return a.root
}

func (a *addCmds) addRoot() {
	a.root = &cobra.Command{
		Use:   "add [app|forward]",
		Short: "adds a new rule",
	}
}

func (a *addCmds) addApp() {
	a.app = &cobra.Command{
		Use:   "app [route-id] [remote-pk]",
		Short: "adds an app rule",
		Args:  cobra.MinimumNArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			id, err := strconv.ParseUint(args[0], 10, 32)
			catch(err)

			pk := cipher.PubKey{}
			catch(pk.Set(args[1]))

			rule := routing.AppRule(time.Now().Add(a.appFlags.expireAfter),
				routing.RouteID(id), pk, a.appFlags.remotePort, a.appFlags.localPort)

			resID, err := client().AddRoutingRule(rule)
			catch(err)

			fmt.Println("id: ", resID)
		},
	}

	a.bindAddAppFlags()
}

func (a *addCmds) bindAddAppFlags() {
	a.app.Flags().DurationVar(&a.appFlags.expireAfter, "expire", 10000*time.Hour,
		"duration after which rule will expire")
	a.app.Flags().Uint16Var(&a.appFlags.localPort, "local-port", 80,
		"local port of the rule")
	a.app.Flags().Uint16Var(&a.appFlags.remotePort, "remote-port", 80,
		"remote port of the rule")
}

func (a *addCmds) addForward() {
	a.forward = &cobra.Command{
		Use:   "forward [next-route-id] [next-transport-id]",
		Short: "adds a forward rule",
		Args:  cobra.MinimumNArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			routeID, err := strconv.ParseUint(args[0], 10, 32)
			catch(err)

			nTrID, err := uuid.Parse(args[1])
			catch(err)

			rule := routing.ForwardRule(time.Now().Add(a.forwardFlags.expireAfter),
				routing.RouteID(routeID), nTrID)

			id, err := client().AddRoutingRule(rule)
			catch(err)

			fmt.Println("id: ", id)
		},
	}

	a.bindAddForwardFlags()
}

func (a *addCmds) bindAddForwardFlags() {
	a.forward.Flags().DurationVar(&a.forwardFlags.expireAfter, "expire", 10000*time.Hour,
		"duration after which rule will expire")
}
