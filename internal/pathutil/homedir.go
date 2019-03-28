package pathutil

import (
	"os"
	"runtime"
)

// HomeDir obtains the path to the user's home directory via ENVs.
// SRC: https://github.com/spf13/viper/blob/80ab6657f9ec7e5761f6603320d3d58dfe6970f6/util.go#L144-L153
func HomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
