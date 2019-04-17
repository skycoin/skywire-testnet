package commands

import (
	"encoding/json"
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

	"github.com/skycoin/skywire/internal/pathutil"
	"github.com/skycoin/skywire/pkg/node"
)

const configEnv = "SW_CONFIG"

var (
	syslogAddr string
	tag        string
)

var rootCmd = &cobra.Command{
	Use:   "skywire-node [config-path]",
	Short: "App Node for skywire",
	Run: func(_ *cobra.Command, args []string) {
		configPath := pathutil.FindConfigPath(args, 0, configEnv, pathutil.NodeDefaults())

		logger := logging.MustGetLogger(tag)

		if syslogAddr != "none" {
			hook, err := logrus_syslog.NewSyslogHook("udp", syslogAddr, syslog.LOG_INFO, tag)
			if err != nil {
				logger.Error("Unable to connect to syslog daemon")
			} else {
				logging.AddHook(hook)
			}
		}

		file, err := os.Open(configPath)
		if err != nil {
			logger.Fatalf("Failed to open config: %s", err)
		}

		conf := &node.Config{}
		if err := json.NewDecoder(file).Decode(&conf); err != nil {
			logger.Fatalf("Failed to decode %s: %s", configPath, err)
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

		ch := make(chan os.Signal, 2)
		signal.Notify(ch, []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}...)
		<-ch
		go func() {
			select {
			case <-time.After(10 * time.Second):
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
	rootCmd.Flags().StringVarP(&tag, "tag", "", "route-finder", "logging tag")
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
