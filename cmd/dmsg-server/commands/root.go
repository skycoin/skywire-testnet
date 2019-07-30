package commands

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"log/syslog"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	logrussyslog "github.com/sirupsen/logrus/hooks/syslog"
	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/spf13/cobra"
)

var (
	metricsAddr  string
	syslogAddr   string
	tag          string
	cfgFromStdin bool
)

// Config is a dmsg-server config
type Config struct {
	PubKey        cipher.PubKey `json:"public_key"`
	SecKey        cipher.SecKey `json:"secret_key"`
	Discovery     string        `json:"discovery"`
	LocalAddress  string        `json:"local_address"`
	PublicAddress string        `json:"public_address"`
	LogLevel      string        `json:"log_level"`
}

var rootCmd = &cobra.Command{
	Use:   "dmsg-server [config.json]",
	Short: "DMSG Server for skywire",
	Run: func(_ *cobra.Command, args []string) {
		// Config
		configFile := "config.json"
		if len(args) > 0 {
			configFile = args[0]
		}
		conf := parseConfig(configFile)

		// Logger
		logger := logging.MustGetLogger(tag)
		logLevel, err := logging.LevelFromString(conf.LogLevel)
		if err != nil {
			log.Fatal("Failed to parse LogLevel: ", err)
		}
		logging.SetLevel(logLevel)

		if syslogAddr != "" {
			hook, err := logrussyslog.NewSyslogHook("udp", syslogAddr, syslog.LOG_INFO, tag)
			if err != nil {
				logger.Fatalf("Unable to connect to syslog daemon on %v", syslogAddr)
			}
			logging.AddHook(hook)
		}

		// Metrics
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(metricsAddr, nil); err != nil {
				logger.Println("Failed to start metrics API:", err)
			}
		}()

		l, err := net.Listen("tcp", conf.LocalAddress)
		if err != nil {
			logger.Fatalf("Error listening on %s: %v", conf.LocalAddress, err)
		}

		// Start
		srv, err := dmsg.NewServer(conf.PubKey, conf.SecKey, conf.PublicAddress, l, disc.NewHTTP(conf.Discovery))
		if err != nil {
			logger.Fatalf("Error creating DMSG server instance: %v", err)
		}

		log.Fatal(srv.Serve())
	},
}

func init() {
	rootCmd.Flags().StringVarP(&metricsAddr, "metrics", "m", ":2121", "address to bind metrics API to")
	rootCmd.Flags().StringVar(&syslogAddr, "syslog", "", "syslog server address. E.g. localhost:514")
	rootCmd.Flags().StringVar(&tag, "tag", "dmsg-server", "logging tag")
	rootCmd.Flags().BoolVarP(&cfgFromStdin, "stdin", "i", false, "read configuration from STDIN")
}

func parseConfig(configFile string) *Config {
	var rdr io.Reader
	var err error
	if !cfgFromStdin {
		rdr, err = os.Open(filepath.Clean(configFile))
		if err != nil {
			log.Fatalf("Failed to open config: %s", err)
		}
	} else {
		rdr = bufio.NewReader(os.Stdin)
	}

	conf := &Config{}
	if err := json.NewDecoder(rdr).Decode(&conf); err != nil {
		log.Fatalf("Failed to decode %s: %s", rdr, err)
	}

	return conf
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
