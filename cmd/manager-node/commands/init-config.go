package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/manager"
)

const (
	homeMode  = "HOME"
	localMode = "LOCAL"
)

var initConfigModes = []string{homeMode, localMode}

var (
	output  string
	replace bool
	mode    string
)

func init() {
	rootCmd.AddCommand(initConfigCmd)

	initConfigCmd.Flags().StringVarP(&output, "output", "o", defaultConfigPaths[0], "path of output config file.")
	initConfigCmd.Flags().BoolVarP(&replace, "replace", "r", false, "whether to allow rewrite of a file that already exists.")
	initConfigCmd.Flags().StringVarP(&mode, "mode", "m", homeMode, fmt.Sprintf("config generation mode. Valid values: %v", initConfigModes))
}

var initConfigCmd = &cobra.Command{
	Use:   "init-config",
	Short: "generates a configuration file",
	Run: func(_ *cobra.Command, _ []string) {
		output, err := filepath.Abs(output)
		if err != nil {
			log.WithError(err).Fatalln("invalid output provided")
		}
		var conf manager.Config
		switch mode {
		case homeMode:
			conf = manager.GenerateHomeConfig()
		case localMode:
			conf = manager.GenerateLocalConfig()
		default:
			log.Fatalln("invalid mode:", mode)
		}
		raw, err := json.MarshalIndent(conf, "", "  ")
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
	},
}
