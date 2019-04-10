package app

import (
	"encoding/gob"
	"os"
)

func init() {
	gob.Register(Config{})
}

// Consts of lfe.
const (
	ProtocolVersion = "0.0.1"
	ConfigCmdName   = "sw-config"
)

var (
	_config *Config
)

// Config defines configuration parameters for App
type Config struct {
	AppName         string
	AppVersion      string
	ProtocolVersion string
}

// Init initiates the app.
func Init(appName, appVersion string) {

	_config = &Config{
		AppName:         appName,
		AppVersion:      appVersion,
		ProtocolVersion: ProtocolVersion,
	}

	// If command is of format: "<exec> sw-config"
	if len(os.Args) == 2 && os.Args[1] == ConfigCmdName {
		if appName != os.Args[0] {
			log.Fatalf("Registered name '%s' does not match executable name '%s'.", appName, os.Args[0])
		}
		if err := gob.NewEncoder(os.Stdout).Encode(_config); err != nil {
			log.Fatalf("Failed to write to stdout: %s", err.Error())
		}
		os.Exit(0)
	}

}
