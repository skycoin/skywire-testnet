package commands

import (
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/node"
	"github.com/skycoin/skywire/pkg/util/pathutil"
)

const configEnv = "SW_CONFIG"

var log = logging.MustGetLogger("skywire-node")

var rootCmd = &cobra.Command{
	Use:   "skywire-node [config-path]",
	Short: "App Node for skywire",
	Run: func(_ *cobra.Command, args []string) {
		configPath := pathutil.FindConfigPath(args, 0, configEnv, pathutil.NodeDefaults())

		file, err := os.Open(configPath)
		if err != nil {
			log.Fatalf("Failed to open config: %s", err)
		}

		conf := &node.Config{}
		if err := json.NewDecoder(file).Decode(&conf); err != nil {
			log.Fatalf("Failed to decode %s: %s", configPath, err)
		}

		node, err := node.NewNode(conf)
		if err != nil {
			log.Fatal("Failed to initialise node: ", err)
		}

		go func() {
			if err := node.Start(); err != nil {
				log.Fatal("Failed to start node: ", err)
			}
		}()

		ch := make(chan os.Signal, 2)
		signal.Notify(ch, []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}...)
		<-ch
		go func() {
			select {
			case <-time.After(10 * time.Second):
				log.Fatal("Timeout reached: terminating")
			case s := <-ch:
				log.Fatalf("Received signal %s: terminating", s)
			}
		}()

		if err := node.Close(); err != nil {
			if !strings.Contains(err.Error(), "closed") {
				log.Fatal("Failed to close node: ", err)
			}
		}
	},
	Version: node.Version,
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
