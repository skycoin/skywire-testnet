package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/syslog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/metrics"
	"github.com/skycoin/skywire/pkg/setup"
)

var (
	metricsAddr  string
	syslogAddr   string
	tag          string
	cfgFromStdin bool
)

var rootCmd = &cobra.Command{
	Use:   "setup-node [config.json]",
	Short: "Route Setup Node for skywire",
	Run: func(_ *cobra.Command, args []string) {

		logger := logging.MustGetLogger(tag)
		if syslogAddr != "" {
			hook, err := logrus_syslog.NewSyslogHook("udp", syslogAddr, syslog.LOG_INFO, tag)
			if err != nil {
				logger.Fatalf("Unable to connect to syslog daemon on %v", syslogAddr)
			}
			logging.AddHook(hook)
		}

		var rdr io.Reader
		var err error

		if !cfgFromStdin {
			configFile := "config.json"

			if len(args) > 0 {
				configFile = args[0]
			}
			rdr, err = os.Open(configFile)
			if err != nil {
				log.Fatalf("Failed to open config: %s", err)
			}
		} else {
			logger.Info("Reading config from STDIN")
			rdr = bufio.NewReader(os.Stdin)
		}

		conf := &setup.Config{}
		if err := json.NewDecoder(rdr).Decode(&conf); err != nil {
			log.Fatalf("Failed to decode %s: %s", rdr, err)
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
	rootCmd.Flags().BoolVarP(&cfgFromStdin, "stdin", "i", false, "read config from STDIN")
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
