package commands

import (
	"context"
	"encoding/json"
	"log"
	"log/syslog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/internal/metrics"
	"github.com/skycoin/skywire/pkg/setup"
)

var (
	metricsAddr string
	syslogAddr  string
	tag         string
)

var rootCmd = &cobra.Command{
	Use:   "setup-node [config.json]",
	Short: "Route Setup Node for skywire",
	Run: func(_ *cobra.Command, args []string) {
		configFile := "config.json"
		if len(args) > 0 {
			configFile = args[0]
		}
		conf := parseConfig(configFile)

		logger := logging.MustGetLogger(tag)
		if syslogAddr != "" {
			hook, err := logrus_syslog.NewSyslogHook("udp", syslogAddr, syslog.LOG_INFO, tag)
			if err != nil {
				logger.Fatalf("Unable to connect to syslog daemon on %v", syslogAddr)
			}
			logging.AddHook(hook)
		}

		sn, err := setup.NewNode(conf, metrics.NewPrometheus("setupnode"))
		if err != nil {
			logger.Fatal("Failed to setup Node: ", err)
		}

		go func() {
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(metricsAddr, nil); err != nil {
				logger.Println("Failed to start metrics API:", err)
			}
		}()

		logger.Fatal(sn.Serve(context.Background()))
	},
}

func init() {
	rootCmd.Flags().StringVarP(&metricsAddr, "metrics", "m", ":2121", "address to bind metrics API to")
	rootCmd.Flags().StringVar(&syslogAddr, "syslog", "", "syslog server address. E.g. localhost:514")
	rootCmd.Flags().StringVar(&tag, "tag", "setup-node", "logging tag")
}

func parseConfig(path string) *setup.Config {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open config: %s", err)
	}

	conf := &setup.Config{}
	if err := json.NewDecoder(file).Decode(&conf); err != nil {
		log.Fatalf("Failed to decode %s: %s", path, err)
	}

	return conf
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
