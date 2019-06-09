package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/spf13/cobra"

	"net/http"
	_ "net/http/pprof" //no_lint

	"github.com/pkg/profile"
	"github.com/skycoin/skywire/pkg/node"
	"github.com/skycoin/skywire/pkg/util/pathutil"
)

const configEnv = "SW_CONFIG"
const defaultShutdownTimeout = node.Duration(10 * time.Second)

var (
	syslogAddr   string
	tag          string
	cfgFromStdin bool
	profileMode  string
	pport        string
)

var rootCmd = &cobra.Command{
	Use:   "skywire-node [config-path]",
	Short: "App Node for skywire",
	Run: func(_ *cobra.Command, args []string) {

		profilePath := profile.ProfilePath("./logs/" + tag)
		switch profileMode {
		case "cpu":
			defer profile.Start(profilePath, profile.CPUProfile).Stop()
		case "mem":
			defer profile.Start(profilePath, profile.MemProfile).Stop()
		case "mutex":
			defer profile.Start(profilePath, profile.MutexProfile).Stop()
		case "block":
			defer profile.Start(profilePath, profile.BlockProfile).Stop()
		case "trace":
			defer profile.Start(profilePath, profile.TraceProfile).Stop()
		case "http":
			go func() {
				log.Println(http.ListenAndServe(fmt.Sprintf("localhost:%v", pport), nil))
			}()
		default:
			// do nothing
		}

		logger := logging.MustGetLogger(tag)

		if syslogAddr != "none" {
			hook, err := logrus_syslog.NewSyslogHook("udp", syslogAddr, syslog.LOG_INFO, tag)
			if err != nil {
				logger.Error("Unable to connect to syslog daemon")
			} else {
				logging.AddHook(hook)
			}
		}

		var rdr io.Reader
		var err error
		if !cfgFromStdin {
			configPath := pathutil.FindConfigPath(args, 0, configEnv, pathutil.NodeDefaults())
			rdr, err = os.Open(configPath)
			if err != nil {
				logger.Fatalf("Failed to open config: %s", err)
			}
		} else {
			logger.Info("Reading config from STDIN")
			rdr = bufio.NewReader(os.Stdin)
		}

		conf := &node.Config{}
		if err := json.NewDecoder(rdr).Decode(&conf); err != nil {
			logger.Fatalf("Failed to decode %s: %s", rdr, err)
		}

		node, err := node.NewNode(conf)
		if err != nil {
			logger.Fatal("Failed to initialise node: ", err)
		}

		go func() {
			if err := node.Start(); err != nil {
				logger.Fatal("Failed to start node: ", err)
			}
		}()

		if conf.ShutdownTimeout == 0 {
			conf.ShutdownTimeout = defaultShutdownTimeout
		}
		ch := make(chan os.Signal, 2)
		signal.Notify(ch, []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}...)
		<-ch
		go func() {
			select {
			case <-time.After(time.Duration(conf.ShutdownTimeout)):
				logger.Fatal("Timeout reached: terminating")
			case s := <-ch:
				logger.Fatalf("Received signal %s: terminating", s)
			}
		}()

		if err := node.Close(); err != nil {
			if !strings.Contains(err.Error(), "closed") {
				logger.Fatal("Failed to close node: ", err)
			}
		}
	},
	Version: node.Version,
}

func init() {
	rootCmd.Flags().StringVarP(&syslogAddr, "syslog", "", "none", "syslog server address. E.g. localhost:514")
	rootCmd.Flags().StringVarP(&tag, "tag", "", "skywire", "logging tag")
	rootCmd.Flags().BoolVarP(&cfgFromStdin, "stdin", "i", false, "read config from STDIN")
	rootCmd.Flags().StringVarP(&profileMode, "pprof", "p", "none", "enable profiling with pprof. Mode:  none or one of: [cpu, mem, mutex, block, trace, http]")
	rootCmd.Flags().StringVarP(&pport, "pport", "", "6060", "port for http-mode of pprof")
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
