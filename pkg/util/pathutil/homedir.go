package pathutil

import (
	"fmt"
	"github.com/skycoin/skywire/pkg/cipher"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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

// NodeDir returns a path to a directory used to store specific node configuration. Such dir is ~/.skywire/{PK}
func NodeDir(pk cipher.PubKey) string {
	return filepath.Join(HomeDir(),".skycoin","skywire",pk.String())
}

// EnsureDir attempts to create given directory, panics if it fails to do so
func EnsureDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0644)
		if err != nil {
			panic(err)
		}
	}
}

// AtomicWriteFile creates a temp file in which to write data, then calls syscall.Rename to swap it and write it on
// filename for an atomic write. On failure temp file is removed and panics.
func AtomicWriteFile(filename string, data []byte) {
	fmt.Println("got filename: ", filename)
	dir, name := path.Split(filename)
	f, err := ioutil.TempFile(dir, name)
	if err != nil {
		panic(err)
	}

	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if permErr := os.Chmod(f.Name(), 0644); err == nil {
		err = permErr
	}
	if err == nil {
		err = os.Rename(f.Name(), filename)
	}

	if err != nil {
		os.Remove(f.Name())
	}
	panic(err)
}

// AtomicAppendToFile calls AtomicWriteFile but appends new data to destiny file
func AtomicAppendToFile(filename string, data []byte) {
	fmt.Println("got filename: ", filename)
	oldFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	AtomicWriteFile(filename, append(oldFile, data...))
}

