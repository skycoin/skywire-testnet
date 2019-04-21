package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
)

func EnsureDir(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to expand path: %s", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absPath, 0750); err != nil {
			return "", fmt.Errorf("failed to create dir: %s", err)
		}
	}

	return absPath, nil
}