package commands

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/skycoin/skywire/internal/pathutil"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/node"
)

var rootCmd = &cobra.Command{
	Use:   "skywire-node [skywire.json]",
	Short: "App Node for skywire",
	Run: func(_ *cobra.Command, args []string) {
		var configFile string
		if len(args) > 0 {
			configFile = args[0]
		} else if conf, ok := os.LookupEnv("SKYWIRE_CONFIG"); ok {
			configFile = conf
		} else {
			conf, err := pathutil.Find("skywire.json")
			if err != nil {
				log.Fatalln(err)
			}
			configFile = conf
		}

		log.Println("using conf file at: ", configFile)

		file, err := os.Open(configFile)
		if err != nil {
			log.Fatalf("Failed to open config: %s", err)
		}

		conf := &node.Config{}
		if err := json.NewDecoder(file).Decode(&conf); err != nil {
			log.Fatalf("Failed to decode %s: %s", configFile, err)
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
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
