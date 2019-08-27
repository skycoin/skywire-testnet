package pathutil

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/skycoin/skycoin/src/util/logging"
)

var log = logging.MustGetLogger("pathutil")

// ConfigLocationType describes a config path's location type.
type ConfigLocationType string

const (
	// WorkingDirLoc represents the default working directory location for a configuration file.
	WorkingDirLoc = ConfigLocationType("WD")

	// HomeLoc represents the default home folder location for a configuration file.
	HomeLoc = ConfigLocationType("HOME")

	// LocalLoc represents the default /usr/local location for a configuration file.
	LocalLoc = ConfigLocationType("LOCAL")
)

// String implements fmt.Stringer for ConfigLocationType.
func (t ConfigLocationType) String() string {
	return string(t)
}

// Set implements pflag.Value for ConfigLocationType.
func (t *ConfigLocationType) Set(s string) error {
	*t = ConfigLocationType(s)
	return nil
}

// Type implements pflag.Value for ConfigLocationType.
func (t ConfigLocationType) Type() string {
	return "pathutil.ConfigLocationType"
}

// AllConfigLocationTypes returns all valid config location types.
func AllConfigLocationTypes() []ConfigLocationType {
	return []ConfigLocationType{
		WorkingDirLoc,
		HomeLoc,
		LocalLoc,
	}
}

// ConfigPaths contains a map of configuration paths, based on ConfigLocationTypes.
type ConfigPaths map[ConfigLocationType]string

// String implements fmt.Stringer for ConfigPaths.
func (dp ConfigPaths) String() string {
	raw, err := json.MarshalIndent(dp, "", "\t")
	if err != nil {
		log.Fatalf("cannot marshal default paths: %s", err.Error())
	}
	return string(raw)
}

// Get obtains a path stored under given configuration location type.
func (dp ConfigPaths) Get(cpType ConfigLocationType) string {
	if path, ok := dp[cpType]; ok {
		return path
	}
	log.Fatalf("invalid config type '%s' provided. Valid types: %v", cpType, AllConfigLocationTypes())
	return ""
}

// NodeDefaults returns the default config paths for skywire-visor.
func NodeDefaults() ConfigPaths {
	paths := make(ConfigPaths)
	if wd, err := os.Getwd(); err == nil {
		paths[WorkingDirLoc] = filepath.Join(wd, "skywire-config.json")
	}
	paths[HomeLoc] = filepath.Join(HomeDir(), ".skycoin/skywire/skywire-config.json")
	paths[LocalLoc] = "/usr/local/skycoin/skywire/skywire-config.json"
	return paths
}

// HypervisorDefaults returns the default config paths for hypervisor.
func HypervisorDefaults() ConfigPaths {
	paths := make(ConfigPaths)
	if wd, err := os.Getwd(); err == nil {
		paths[WorkingDirLoc] = filepath.Join(wd, "hypervisor-config.json")
	}
	paths[HomeLoc] = filepath.Join(HomeDir(), ".skycoin/hypervisor/hypervisor-config.json")
	paths[LocalLoc] = "/usr/local/skycoin/hypervisor/hypervisor-config.json"
	return paths
}

// FindConfigPath is used by a service to find a config file path in the following order:
// - From CLI argument.
// - From ENV.
// - From a list of default paths.
// If argsIndex < 0, searching from CLI arguments does not take place.
func FindConfigPath(args []string, argsIndex int, env string, defaults ConfigPaths) string {
	if argsIndex >= 0 && len(args) > argsIndex {
		path := args[argsIndex]
		log.Infof("using args[%d] as config path: %s", argsIndex, path)
		return path
	}
	if env != "" {
		if path, ok := os.LookupEnv(env); ok {
			log.Infof("using $%s as config path: %s", env, path)
			return path
		}
	}
	log.Debugf("config path is not explicitly specified, trying default paths...")
	for i, cpType := range []ConfigLocationType{WorkingDirLoc, HomeLoc, LocalLoc} {
		path, ok := defaults[cpType]
		if !ok {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			log.Debugf("- [%d/%d] '%s' cannot be accessed: %s", i+1, len(defaults), path, err.Error())
		} else {
			log.Debugf("- [%d/%d] '%s' is found", i+1, len(defaults), path)
			log.Printf("using fallback config path: %s", path)
			return path
		}
	}
	log.Fatalf("config not found in any of the following paths: %s", defaults.String())
	return ""
}

// WriteJSONConfig is used by config file generators.
// 'output' specifies the path to save generated config files.
// 'replace' is true if replacing files is allowed.
func WriteJSONConfig(conf interface{}, output string, replace bool) {
	raw, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		log.WithError(err).Fatal("unexpected error, report to dev")
	}
	if _, err := os.Stat(output); !replace && err == nil {
		log.Fatalf("file %s already exists, stopping as 'replace,r' flag is not set", output)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0750); err != nil {
		log.WithError(err).Fatalln("failed to create output directory")
	}
	if err := ioutil.WriteFile(output, raw, 0744); err != nil {
		log.WithError(err).Fatalln("failed to write file")
	}
	log.Infof("Wrote %d bytes to %s\n%s", len(raw), output, string(raw))
}
