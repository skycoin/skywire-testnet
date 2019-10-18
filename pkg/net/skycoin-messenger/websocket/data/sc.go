package data

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/SkycoinProject/skywire/pkg/net/skycoin-messenger/factory"
)

var (
	keysPath  string
	keys      = make(map[string]*factory.SeedConfig)
	keysMutex = &sync.Mutex{}
)

func walkFunc(path string, info os.FileInfo, err error) (e error) {
	if err != nil || info.IsDir() {
		return
	}
	sc, e := factory.ReadSeedConfig(path)
	if e != nil {
		return e
	}
	keys[sc.PublicKey] = sc
	return
}

// Load keys from path and save path for further calls
func InitData(path string) (err error) {
	keysMutex.Lock()
	err = filepath.Walk(path, walkFunc)
	keysPath = path
	keysMutex.Unlock()
	return
}

// Load keys from path
func GetData() (result map[string]*factory.SeedConfig, err error) {
	if len(keysPath) < 1 {
		err = errors.New("keysPath can not be empty")
		return
	}

	keysMutex.Lock()
	k := keys
	err = filepath.Walk(keysPath, walkFunc)
	keysMutex.Unlock()
	if err != nil {
		keys = k
	}
	result = keys
	return
}

// Get loaded keys
func GetKeys() (result map[string]*factory.SeedConfig, err error) {
	if len(keysPath) < 1 {
		err = errors.New("keysPath can not be empty")
		return
	}

	keysMutex.Lock()
	result = keys
	keysMutex.Unlock()
	return
}

// Create key and save to the path
func AddKey() (result map[string]*factory.SeedConfig, err error) {
	if len(keysPath) < 1 {
		err = errors.New("keysPath can not be empty")
	}

	sc := factory.NewSeedConfig()
	err = factory.WriteSeedConfig(sc, filepath.Join(keysPath, sc.PublicKey))
	if err != nil {
		return
	}
	return GetData()
}

func AddKeyToReg() (sc *factory.SeedConfig, err error) {
	if len(keysPath) < 1 {
		err = errors.New("keysPath can not be empty")
	}
	sc = factory.NewSeedConfig()
	err = factory.WriteSeedConfig(sc, filepath.Join(keysPath, sc.PublicKey))
	if err != nil {
		return
	}
	keysMutex.Lock()
	keys[sc.PublicKey] = sc
	keysMutex.Unlock()
	return
}
