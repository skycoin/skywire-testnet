package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/pkg/discovery"
	"github.com/skycoin/skywire/pkg/net/util"

	"net/http"
	_ "net/http/pprof"
)

var (
	address  string
	webDir   string
	webPort  string
	seedPath string

	ipDBPath string
	confPath string

	version bool

	showSQL     bool
	sqlLogLevel string

	logLevel       string
	parsedLogLevel log.Level = log.DebugLevel
	disableLog     bool
	logFilename    string

	profile     bool
	profileAddr string
)

func parseFlags() {
	var dir = "/src/github.com/skycoin/skywire/pkg/net/skycoin-messenger/monitor/web/dist-discovery"
	flag.StringVar(&address, "address", ":5999", "address to listen on")
	flag.StringVar(&webDir, "web-dir", filepath.Join(os.Getenv("GOPATH"), dir), "monitor web page")
	flag.StringVar(&webPort, "web-port", ":8000", "monitor web page port")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "discovery", "keys.json"), "path to save seed info")
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&ipDBPath, "ipdb-path", filepath.Join(dir, "ip.db"), "ip db file path")
	flag.StringVar(&confPath, "conf-path", filepath.Join(file.UserHome(), ".skywire", "discovery", "conf.json"), "config file path")
	flag.BoolVar(&version, "v", false, "print current version")
	flag.BoolVar(&showSQL, "show-sql", false, "print sql statements for log statements >= INFO")
	flag.StringVar(&sqlLogLevel, "sql-log-level", "off", "xorm sql log level, choices are debug, info, warn, error, off")

	flag.StringVar(&logLevel, "log-level", "debug", "general log level, choices are debug, info, warn, error, fatal, panic")
	flag.BoolVar(&disableLog, "disable-log", false, "disable general logging")
	flag.StringVar(&logFilename, "log-filename", "", "general logging writes to this file, if set")

	flag.BoolVar(&profile, "profile", false, "enable http/pprof on profile-port")
	flag.StringVar(&profileAddr, "profile-addr", "localhost:6060", "http/pprof listen address")

	flag.Parse()

	if logLevel != "" {
		var err error
		parsedLogLevel, err = log.ParseLevel(logLevel)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func initLogger() {
	log.SetOutput(os.Stdout)

	log.SetLevel(parsedLogLevel)

	if disableLog {
		fmt.Println("Disabling log output")
		log.SetOutput(ioutil.Discard)
	}

	log.SetFormatter(&log.TextFormatter{})

	if logFilename != "" {
		f, err := os.OpenFile("discovery.log", os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		log.SetOutput(f)
	}
}

func initProfiler() {
	log.Infof("profile enabled on %s", profileAddr)
	go func() {
		if err := http.ListenAndServe(profileAddr, nil); err != nil {
			log.WithError(err).Error("Failed to start http profiler")
		}
	}()
}

func main() {
	parseFlags()
	if version {
		fmt.Println(discovery.Version)
		return
	}

	initLogger()

	if profile {
		initProfiler()
	}

	var err error
	err = util.IPLocator.Init(ipDBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer util.IPLocator.Close()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	d := discovery.New(seedPath, address, webPort, webDir)
	d.SQLLogLevel = sqlLogLevel
	d.ShowSQL = showSQL
	err = d.Start()
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("listen on %s", address)

	defer d.Close()

	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
	}
}
