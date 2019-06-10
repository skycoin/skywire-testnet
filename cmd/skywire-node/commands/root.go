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

type runCfg struct {
	syslogAddr   string
	tag          string
	cfgFromStdin bool
	profileMode  string
	port         string
	args         []string

	profileStop func()
	logger      *logging.Logger
	conf        node.Config
	node        *node.Node
}

var cfg *runCfg

func (cfg *runCfg) startProfiler() *runCfg {
	log.Println("profile mode == ", cfg.profileMode)
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
	default:
	}
	cfg.profileStop = profile.Start(profile.ProfilePath("./logs/"+cfg.tag), option).Stop
	return cfg
}

func (cfg *runCfg) startLogger() *runCfg {
	cfg.logger = logging.MustGetLogger(cfg.tag)

	if cfg.syslogAddr != "none" {
		hook, err := logrus_syslog.NewSyslogHook("udp", cfg.syslogAddr, syslog.LOG_INFO, cfg.tag)
		if err != nil {
			cfg.logger.Error("Unable to connect to syslog daemon")
		} else {
			logging.AddHook(hook)
		}
	}
	return cfg
}

func (cfg *runCfg) readConfig() *runCfg {
	var rdr io.Reader
	var err error
	if !cfg.cfgFromStdin {
		configPath := pathutil.FindConfigPath(cfg.args, 0, configEnv, pathutil.NodeDefaults())
		rdr, err = os.Open(configPath)
		if err != nil {
			cfg.logger.Fatalf("Failed to open config: %s", err)
		}
	} else {
		cfg.logger.Info("Reading config from STDIN")
		rdr = bufio.NewReader(os.Stdin)
	}

	cfg.conf = node.Config{}
	if err := json.NewDecoder(rdr).Decode(&cfg.conf); err != nil {
		cfg.logger.Fatalf("Failed to decode %s: %s", rdr, err)
	}
	return cfg
}

func (cfg *runCfg) runNode() *runCfg {
	node, err := node.NewNode(&cfg.conf)
	if err != nil {
		cfg.logger.Fatal("Failed to initialise node: ", err)
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

// waitOsSignalsAndDelay
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

var rootCmd = &cobra.Command{
	Use:   "skywire-node [config-path]",
	Short: "App Node for skywire",
	Run: func(_ *cobra.Command, args []string) {
		cfg.args = args

		cfg.startProfiler().
			startLogger().
			readConfig().
			runNode().
			waitOsSignals().
			stopNode()
	},
	Version: node.Version,
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
