package commands

import (
	"bufio"
	"encoding/json"
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

	"github.com/skycoin/skywire/pkg/node"
	"github.com/skycoin/skywire/pkg/util/pathutil"
)

const configEnv = "SW_CONFIG"
const defaultShutdownTimeout = node.Duration(10 * time.Second)

var (
	syslogAddr   string
	tag          string
	cfgFromStdin bool
)

var rootCmd = &cobra.Command{
	Use:   "skywire-node [config-path]",
	Short: "App Node for skywire",
	Run: func(_ *cobra.Command, args []string) {

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
			logger.Fatal("Failed to initialize node: ", err)
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
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
