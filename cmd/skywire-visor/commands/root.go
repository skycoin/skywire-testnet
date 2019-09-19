package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"log/syslog"
	"net/http"
	_ "net/http/pprof" // used for HTTP profiling
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/profile"
	logrussyslog "github.com/sirupsen/logrus/hooks/syslog"
	"github.com/SkycoinProject/skycoin/src/util/logging"
	"github.com/spf13/cobra"
	"github.com/SkycoinProject/skywire-mainnet/internal/utclient"
	"github.com/SkycoinProject/skywire-mainnet/pkg/util/pathutil"
	"github.com/SkycoinProject/skywire-mainnet/pkg/visor"
)

const configEnv = "SW_CONFIG"
const defaultShutdownTimeout = visor.Duration(10 * time.Second)

type runCfg struct {
	syslogAddr   string
	tag          string
	cfgFromStdin bool
	profileMode  string
	port         string
	args         []string

	profileStop  func()
	logger       *logging.Logger
	masterLogger *logging.MasterLogger
	conf         visor.Config
	node         *visor.Node
}

var cfg *runCfg

var rootCmd = &cobra.Command{
	Use:   "skywire-visor [config-path]",
	Short: "Visor for skywire",
	Run: func(_ *cobra.Command, args []string) {
		cfg.args = args

		cfg.startProfiler().
			startLogger().
			readConfig().
			runNode().
			waitOsSignals().
			stopNode()
	},
	Version: visor.Version,
}

func init() {
	cfg = &runCfg{}
	rootCmd.Flags().StringVarP(&cfg.syslogAddr, "syslog", "", "none", "syslog server address. E.g. localhost:514")
	rootCmd.Flags().StringVarP(&cfg.tag, "tag", "", "skywire", "logging tag")
	rootCmd.Flags().BoolVarP(&cfg.cfgFromStdin, "stdin", "i", false, "read config from STDIN")
	rootCmd.Flags().StringVarP(&cfg.profileMode, "profile", "p", "none", "enable profiling with pprof. Mode:  none or one of: [cpu, mem, mutex, block, trace, http]")
	rootCmd.Flags().StringVarP(&cfg.port, "port", "", "6060", "port for http-mode of pprof")
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func (cfg *runCfg) startProfiler() *runCfg {
	var option func(*profile.Profile)
	switch cfg.profileMode {
	case "none":
		cfg.profileStop = func() {}
		return cfg
	case "http":
		go func() {
			log.Println(http.ListenAndServe(fmt.Sprintf("localhost:%v", cfg.port), nil))
		}()
		cfg.profileStop = func() {}
		return cfg
	case "cpu":
		option = profile.CPUProfile
	case "mem":
		option = profile.MemProfile
	case "mutex":
		option = profile.MutexProfile
	case "block":
		option = profile.BlockProfile
	case "trace":
		option = profile.TraceProfile
	}
	cfg.profileStop = profile.Start(profile.ProfilePath("./logs/"+cfg.tag), option).Stop
	return cfg
}

func (cfg *runCfg) startLogger() *runCfg {
	cfg.masterLogger = logging.NewMasterLogger()
	cfg.logger = cfg.masterLogger.PackageLogger(cfg.tag)

	if cfg.syslogAddr != "none" {
		hook, err := logrussyslog.NewSyslogHook("udp", cfg.syslogAddr, syslog.LOG_INFO, cfg.tag)
		if err != nil {
			cfg.logger.Error("Unable to connect to syslog daemon:", err)
		} else {
			cfg.masterLogger.AddHook(hook)
			cfg.masterLogger.Out = ioutil.Discard
		}
	}
	return cfg
}

func (cfg *runCfg) readConfig() *runCfg {
	var rdr io.Reader
	var err error
	if !cfg.cfgFromStdin {
		configPath := pathutil.FindConfigPath(cfg.args, 0, configEnv, pathutil.NodeDefaults())
		rdr, err = os.Open(filepath.Clean(configPath))
		if err != nil {
			cfg.logger.Fatalf("Failed to open config: %s", err)
		}
	} else {
		cfg.logger.Info("Reading config from STDIN")
		rdr = bufio.NewReader(os.Stdin)
	}

	cfg.conf = visor.Config{}
	if err := json.NewDecoder(rdr).Decode(&cfg.conf); err != nil {
		cfg.logger.Fatalf("Failed to decode %s: %s", rdr, err)
	}
	fmt.Println("TCP Factory conf:", cfg.conf.TCPTransport)
	return cfg
}

func (cfg *runCfg) runNode() *runCfg {
	node, err := visor.NewNode(&cfg.conf, cfg.masterLogger)
	if err != nil {
		cfg.logger.Fatal("Failed to initialize node: ", err)
	}

	if cfg.conf.Uptime.Tracker != "" {
		uptimeTracker, err := utclient.NewHTTP(cfg.conf.Uptime.Tracker, cfg.conf.Node.StaticPubKey, cfg.conf.Node.StaticSecKey)
		if err != nil {
			cfg.logger.Error("Failed to connect to uptime tracker: ", err)
		} else {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			go func() {
				for range ticker.C {
					ctx := context.Background()
					if err := uptimeTracker.UpdateNodeUptime(ctx); err != nil {
						cfg.logger.Error("Failed to update node uptime: ", err)
					}
				}
			}()
		}
	}

	go func() {
		if err := node.Start(); err != nil {
			cfg.logger.Fatal("Failed to start node: ", err)
		}
	}()

	if cfg.conf.ShutdownTimeout == 0 {
		cfg.conf.ShutdownTimeout = defaultShutdownTimeout
	}
	cfg.node = node
	return cfg
}

func (cfg *runCfg) stopNode() *runCfg {
	defer cfg.profileStop()
	if err := cfg.node.Close(); err != nil {
		if !strings.Contains(err.Error(), "closed") {
			cfg.logger.Fatal("Failed to close node: ", err)
		}
	}
	return cfg
}

func (cfg *runCfg) waitOsSignals() *runCfg {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}...)
	<-ch
	go func() {
		select {
		case <-time.After(time.Duration(cfg.conf.ShutdownTimeout)):
			cfg.logger.Fatal("Timeout reached: terminating")
		case s := <-ch:
			cfg.logger.Fatalf("Received signal %s: terminating", s)
		}
	}()
	return cfg
}
