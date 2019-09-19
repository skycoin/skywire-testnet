package node

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/SkycoinProject/skywire-mainnet/cmd/skywire-cli/internal"
	"github.com/SkycoinProject/skywire-mainnet/pkg/visor"
)

func init() {
	RootCmd.AddCommand(
		lsAppsCmd,
		startAppCmd,
		stopAppCmd,
		setAppAutostartCmd,
		appLogsSinceCmd,
		execCmd,
	)
}

var lsAppsCmd = &cobra.Command{
	Use:   "ls-apps",
	Short: "Lists apps running on the local node",
	Run: func(_ *cobra.Command, _ []string) {
		states, err := rpcClient().Apps()
		internal.Catch(err)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
		_, err = fmt.Fprintln(w, "app\tports\tauto_start\tstatus")
		internal.Catch(err)

		for _, state := range states {
			status := "stopped"
			if state.Status == visor.AppStatusRunning {
				status = "running"
			}
			_, err = fmt.Fprintf(w, "%s\t%s\t%t\t%s\n", state.Name, strconv.Itoa(int(state.Port)), state.AutoStart, status)
			internal.Catch(err)
		}
		internal.Catch(w.Flush())
	},
}

var startAppCmd = &cobra.Command{
	Use:   "start-app <name>",
	Short: "Starts an app of given name",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		internal.Catch(rpcClient().StartApp(args[0]))
		fmt.Println("OK")
	},
}

var stopAppCmd = &cobra.Command{
	Use:   "stop-app <name>",
	Short: "Stops an app of given name",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		internal.Catch(rpcClient().StopApp(args[0]))
		fmt.Println("OK")
	},
}

var setAppAutostartCmd = &cobra.Command{
	Use:   "set-app-autostart <name> (on|off)",
	Short: "Sets the autostart flag for an app of given name",
	Args:  cobra.MinimumNArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		var autostart bool
		switch args[1] {
		case "on":
			autostart = true
		case "off":
			autostart = false
		default:
			internal.Catch(fmt.Errorf("invalid args[1] value: %s", args[1]))
		}
		internal.Catch(rpcClient().SetAutoStart(args[0], autostart))
		fmt.Println("OK")
	},
}

var appLogsSinceCmd = &cobra.Command{
	Use:   "app-logs-since <name> <timestamp>",
	Short: "Gets logs from given app since RFC3339Nano-formated timestamp. \"beginning\" is a special timestamp to fetch all the logs",
	Args:  cobra.MinimumNArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		var t time.Time

		if args[1] == "beginning" {
			t = time.Unix(0, 0)
		} else {
			var err error
			strTime := args[1]
			t, err = time.Parse(time.RFC3339Nano, strTime)
			internal.Catch(err)
		}
		logs, err := rpcClient().LogsSince(t, args[0])
		internal.Catch(err)
		if len(logs) > 0 {
			fmt.Println(logs)
		} else {
			fmt.Println("no logs")
		}
	},
}

var execCmd = &cobra.Command{
	Use:   "exec <command>",
	Short: "Executes the given command",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		out, err := rpcClient().Exec(strings.Join(args, " "))
		internal.Catch(err)
		fmt.Print(string(out))
	},
}
