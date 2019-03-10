package commands

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/node"
)

func init() {
	rootCmd.AddCommand(
		appsCmd,
		startAppCmd,
		stopAppCmd,
		setAppAutostartCmd,
	)
}

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "lists apps running on the node",
	Run: func(_ *cobra.Command, _ []string) {
		states, err := rpcClient().Apps()
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

var startAppCmd = &cobra.Command{
	Use:   "start-app <name>",
	Short: "starts an app of given name",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		catch(rpcClient().StartApp(args[0]))
		fmt.Println("OK")
	},
}

var stopAppCmd = &cobra.Command{
	Use:   "stop-app <name>",
	Short: "stops an app of given name",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		catch(rpcClient().StopApp(args[0]))
		fmt.Println("OK")
	},
}

var setAppAutostartCmd = &cobra.Command{
	Use:   "set-app-autostart <name> (on|off)",
	Short: "sets the autostart flag for an app of given name",
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
		catch(rpcClient().SetAutoStart(args[0], autostart))
		fmt.Println("OK")
	},
}
