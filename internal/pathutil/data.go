package pathutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrDirectoryNotFound = errors.New(fmt.Sprintf(
		"directory not found in any of the following paths: %s", strings.Join(paths(), ", ")))
)

// paths return the paths in which skywire looks for data directories
func paths() []string {
	// if we are unable to find the local directory try next option
	localDir, _ := os.Getwd() // nolint: errcheck

	return []string{
		localDir,
		filepath.Join(HomeDir(), ".skycoin/skywire"),
		"/usr/local/skycoin/skywire",
	}
}

// Find tries to find given directory or file in:
// 1) local app directory
// 2) ${HOME}/.skycoin/skywire/{directory}
// 3) /usr/local/skycoin/skywire/{directory}
// It will return the first path, including given directory if found, by preference.
func Find(name string) (string, error) {
	for _, path := range paths() {
		_, err := os.Stat(filepath.Join(path, name))
		if err == nil {
			return filepath.Join(path, name), nil
		}
	}

	return "", ErrDirectoryNotFound
}
