package node

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/cmd/skywire-cli/internal"
	"github.com/skycoin/skywire/pkg/router"
)

var (
	startAppArgs []string
)

func init() {
	startAppCmd.Flags().StringSliceVarP(&startAppArgs, "args", "a", []string{},
		"args in the form \"arg1,arg2,arg3...\". For flags: \"--flag,value\"")

	RootCmd.AddCommand(
		lsAppsCmd,
		startAppCmd,
		stopAppCmd,
		lsProcsCmd,
	)
}

//
var lsAppsCmd = &cobra.Command{
	Use:   "ls-apps",
	Short: "Lists apps available in the node to be run",
	Run: func(_ *cobra.Command, _ []string) {
		appMetas, err := rpcClient().Apps()
		internal.Catch(err)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
		_, err = fmt.Fprintln(w, "name\tversion\tprotocol_version")
		internal.Catch(err)

		for _, meta := range appMetas {
			_, err = fmt.Fprintf(w, "%s\t%s\t%s\n", meta.AppName, meta.AppVersion, meta.ProtocolVersion)
			internal.Catch(err)
		}
		internal.Catch(w.Flush())
	},
}

var startAppCmd = &cobra.Command{
	Use:   "start-app <name> <port>",
	Short: "Starts a process of given app on given port if possible",
	Args:  cobra.MinimumNArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		port := parseUint("port", args[1], 16)

		pid, err := rpcClient().StartProc(args[0], startAppArgs, uint16(port))
		internal.Catch(err, "starting process...")

		fmt.Println("app process started with pid: ", pid)
	},
}

var stopAppCmd = &cobra.Command{
	Use:   "stop-app <name>",
	Short: "Stops an app process of given PID",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		pid := router.ProcID(parseUint("PID", args[0], 16))
		internal.Catch(rpcClient().StopProc(pid))
		fmt.Println("OK")
	},
}

var lsProcsCmd = &cobra.Command{
	Use:   "ls-procs",
	Short: "shows information of app processes ran by node",
	Run: func(_ *cobra.Command, _ []string) {
		infos, err := rpcClient().ListProcs()
		internal.Catch(err)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
		_, err = fmt.Fprintln(w, "name\tversion\tprotocol_version\tworkdir\tbin_loc\targs\tpid\tport")
		internal.Catch(err)

		for _, info := range infos {
			_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\n", info.Meta.AppName, info.AppVersion,
				info.ProtocolVersion, info.WorkDir, info.BinLoc, strings.Join(info.Args, " "),
				info.PID, info.Port)
			internal.Catch(err)
		}
		internal.Catch(w.Flush())
	},
}

//var setAppAutostartCmd = &cobra.Command{
//	Use:   "set-app-autostart <name> (on|off)",
//	Short: "Sets the autostart flag for an app of given name",
//	Args:  cobra.MinimumNArgs(2),
//	Run: func(_ *cobra.Command, args []string) {
//		var autostart bool
//		switch args[1] {
//		case "on":
//			autostart = true
//		case "off":
//			autostart = false
//		default:
//			internal.Catch(fmt.Errorf("invalid args[1] value: %s", args[1]))
//		}
//		internal.Catch(rpcClient().SetAutoStart(args[0], autostart))
//		fmt.Println("OK")
//	},
//}
