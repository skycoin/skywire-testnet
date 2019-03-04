package commands

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/node"
)

type appCmds struct {
	root    *cobra.Command
	list    *cobra.Command
	start   *cobra.Command
	stop    *cobra.Command
	setAuto *cobra.Command
}

func newAppCmds() *cobra.Command {
	a := &appCmds{}
	a.initRoot()
	a.initList()
	a.initStart()
	a.initStop()
	a.initAuto()

	a.root.AddCommand(a.list)
	a.root.AddCommand(a.start)
	a.root.AddCommand(a.stop)
	a.root.AddCommand(a.setAuto)

	return a.root
}

func (a *appCmds) initRoot() {
	a.root = &cobra.Command{
		Use:   "app",
		Short: "app management operations",
	}
}

func (a *appCmds) initList() {
	a.list = &cobra.Command{
		Use:   "list",
		Short: "returns list of apps registered on the node",
		Run: func(_ *cobra.Command, _ []string) {
			states, err := client().Apps()
			catch(err)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
			_, err = fmt.Fprintln(w, "app\tports\tauto_start\tstatus")
			catch(err)

			for _, state := range states {
				status := "stopped"
				if state.Status == node.AppStatusRunning {
					status = "running"
				}

				_, err = fmt.Fprintf(w, "%s\t%s\t%t\t%s\n", state.Name, strconv.Itoa(int(state.Port)), state.AutoStart, status)
				catch(err)
			}

			catch(w.Flush())
		},
	}
}

func (a *appCmds) initStart() {
	a.start = &cobra.Command{
		Use:   "start [name]",
		Short: "starts a Skywire app with given name",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			catch(client().StartApp(args[0]))
			fmt.Println("OK")
		},
	}
}

func (a *appCmds) initStop() {
	a.stop = &cobra.Command{
		Use:   "stop [name]",
		Short: "stops a Skywire app with given name",
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			catch(client().StopApp(args[0]))
			fmt.Println("OK")
		},
	}
}

func (a *appCmds) initAuto() {
	a.setAuto = &cobra.Command{
		Use:   "set-auto [name] [on|off]",
		Short: "sets the auto-start flag on a Skywire app with given name",
		Args:  cobra.MinimumNArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			var autostart bool
			switch args[1] {
			case "on":
				autostart = true
			case "off":
				autostart = false
			default:
				catch(fmt.Errorf("invalid args[1] value: %s", args[1]))
			}
			catch(client().SetAutoStart(args[0], autostart))
			fmt.Println("OK")
		},
	}
}
