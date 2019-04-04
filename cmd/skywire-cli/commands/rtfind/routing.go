package rtfind

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/cmd/skywire-cli/internal"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/node"
	"github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/router"
	"github.com/skycoin/skywire/pkg/routing"
)

// RtFindCmd contains commands that interact with the route finder
var RtFindCmd = &cobra.Command{
	Use:   "rtfind",
	Short: "Commands that interact with the route-finder",
}

func init() {
	RtFindCmd.AddCommand(
		listRulesCmd,
		ruleCmd,
		rmRuleCmd,
		addRuleCmd,
		findRoutesCmd)
}

var listRulesCmd = &cobra.Command{
	Use:   "list-rules",
	Short: "lists the local node's routing rules",
	Run: func(_ *cobra.Command, _ []string) {
		rules, err := internal.RPCClient().RoutingRules()
		internal.Catch(err)

		printRoutingRules(rules...)
	},
}

var ruleCmd = &cobra.Command{
	Use:   "rule <route-id>",
	Short: "returns a routing rule via route ID key",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 32)
		internal.Catch(err)

		rule, err := internal.RPCClient().RoutingRule(routing.RouteID(id))
		internal.Catch(err)

		printRoutingRules(&node.RoutingEntry{Key: rule.RouteID(), Value: rule})
	},
}

var rmRuleCmd = &cobra.Command{
	Use:   "rm-rule <route-id>",
	Short: "removes a routing rule via route ID key",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 32)
		internal.Catch(err)
		internal.Catch(internal.RPCClient().RemoveRoutingRule(routing.RouteID(id)))
		fmt.Println("OK")
	},
}

func init() {
	addRuleCmd.PersistentFlags().DurationVar(&expire, "expire", router.RouteTTL, "duration after which routing rule will expire")
}

var expire time.Duration

var addRuleCmd = &cobra.Command{
	Use:   "add-rule (app <route-id> <remote-pk> <remote-port> <local-port> | fwd <next-route-id> <next-transport-id>)",
	Short: "adds a new routing rule",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			switch rt := args[0]; rt {
			case "app":
				if len(args[0:]) == 4 {
					return nil
				}
				return errors.New("expected 4 args after 'app'")
			case "fwd":
				if len(args[0:]) == 2 {
					return nil
				}
				return errors.New("expected 2 args after 'fwd'")
			}
		}
		return errors.New("expected 'app' or 'fwd' after 'add-rule'")
	},
	Run: func(_ *cobra.Command, args []string) {
		var rule routing.Rule
		switch args[0] {
		case "app":
			var (
				routeID    = routing.RouteID(parseUint("route-id", args[1], 32))
				remotePK   = internal.ParsePK("remote-pk", args[2])
				remotePort = uint16(parseUint("remote-port", args[3], 16))
				localPort  = uint16(parseUint("local-port", args[4], 16))
			)
			rule = routing.AppRule(time.Now().Add(expire), routeID, remotePK, remotePort, localPort)
		case "fwd":
			var (
				nextRouteID = routing.RouteID(parseUint("next-route-id", args[1], 32))
				nextTpID    = internal.ParseUUID("next-transport-id", args[2])
			)
			rule = routing.ForwardRule(time.Now().Add(expire), nextRouteID, nextTpID)
		}
		rIDKey, err := internal.RPCClient().AddRoutingRule(rule)
		internal.Catch(err)
		fmt.Println("Routing Rule Key:", rIDKey)
	},
}

var frAddr string
var frMinHops, frMaxHops uint16

var findRoutesCmd = &cobra.Command{
	Use:   "find-routes <public-key-node-1> <public-key-node-2>",
	Short: "lists available routes between two nodes via route finder service",
	Args:  cobra.MinimumNArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		rfc := client.NewHTTP(frAddr)

		var srcPK, dstPK cipher.PubKey
		internal.Catch(srcPK.Set(args[0]))
		internal.Catch(dstPK.Set(args[1]))

		forward, reverse, err := rfc.PairedRoutes(srcPK, dstPK, frMinHops, frMaxHops)
		internal.Catch(err)

		fmt.Println("forward: ", forward)
		fmt.Println("reverse: ", reverse)
	},
}

func init() {
	findRoutesCmd.Flags().StringVar(&frAddr, "addr", "https://routefinder.skywire.skycoin.net", "address in which to contact route finder service")
	findRoutesCmd.Flags().Uint16Var(&frMinHops, "min-hops", 1, "min hops for the returning routeFinderRoutesCmd")
	findRoutesCmd.Flags().Uint16Var(&frMaxHops, "max-hops", 1000, "max hops for the returning routeFinderRoutesCmd")
}

func printRoutingRules(rules ...*node.RoutingEntry) {
	printAppRule := func(w io.Writer, id routing.RouteID, s *routing.RuleSummary) {
		_, err := fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%s\t%d\t%s\t%s\t%s\n", id, s.Type, s.AppFields.LocalPort,
			s.AppFields.RemotePort, s.AppFields.RemotePK, s.AppFields.RespRID, "-", "-", s.ExpireAt)
		internal.Catch(err)
	}
	printFwdRule := func(w io.Writer, id routing.RouteID, s *routing.RuleSummary) {
		_, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n", id, s.Type, "-",
			"-", "-", "-", s.ForwardFields.NextRID, s.ForwardFields.NextTID, s.ExpireAt)
		internal.Catch(err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
	_, err := fmt.Fprintln(w, "id\ttype\tlocal-port\tremote-port\tremote-pk\tresp-id\tnext-route-id\tnext-transport-id\texpire-at")
	internal.Catch(err)
	for _, rule := range rules {
		if rule.Value.Summary().AppFields != nil {
			printAppRule(w, rule.Key, rule.Value.Summary())
		} else {
			printFwdRule(w, rule.Key, rule.Value.Summary())
		}
	}
	internal.Catch(w.Flush())
}

func parseUint(name, v string, bitSize int) uint64 {
	i, err := strconv.ParseUint(v, 10, bitSize)
	internal.Catch(err, fmt.Sprintf("failed to parse <%s>:", name))
	return i
}
