package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var Global Config

const maxBufferSize = 32000

func Load(configFileName string) error {
	println("Loading configuration file:", configFileName)

	var err error

	file, err := os.Open(configFileName)
	if err != nil {
		return err
	}

	defer file.Close()

	buffer := make([]byte, maxBufferSize)
	n, err := file.Read(buffer)
	if err != nil {
		return err
	}

	Global = Config{}

	err = yaml.Unmarshal(buffer[:n], &Global)
	if err != nil {
		return err
	}

	if Global.Settings.VerifyParsing {
		fmt.Printf("[ Config ]\n")

		for key, app := range Global.Apps {
			fmt.Printf("[ App \"%s\" ]\n", key)
			fmt.Printf("\tPath: %s\n", app.Path)
			fmt.Printf("\tArgs: %v\n", app.Args)
			fmt.Printf("\tDescription: %s\n\n", app.Desc)
			fmt.Printf("\tHelp: %s\n\n", app.Help)
		}

		fmt.Printf("Settings: %+v\n\n", Global.Settings)
	}

	return nil
}

func AppExistsWithName(name string) bool {
	_, exists := Global.Apps[name]
	return exists
}

func GetPathForApp(name string) string {
	return Global.Apps[name].Path
}

func GetPathWithDefaultArgsForApp(name string) []string {
	app := Global.Apps[name]

	tokens := []string{app.Path}

	tokens = append(tokens, app.Args...)

	return tokens
}

func DebugPrintInputEvents() bool {
	return Global.Settings.VerboseInput
}
