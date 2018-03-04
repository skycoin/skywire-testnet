package factory

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/go-bip39"
)

type ConnConfig struct {
	Reconnect     bool
	ReconnectWait time.Duration

	// generate seed, private key and public key for the connection
	// seed config file path
	SeedConfigPath string
	SeedConfig     *SeedConfig

	// context
	Context map[string]string

	UseCrypto RegVersion

	TargetKey cipher.PubKey

	SkipBeforeCallbacks bool

	// callbacks

	FindServiceNodesByKeysCallback func(resp *QueryResp)

	FindServiceNodesByAttributesCallback func(resp *QueryByAttrsResp)

	AppConnectionInitCallback func(resp *AppConnResp) *AppFeedback

	// call after connected to server
	OnConnected func(connection *Connection)
	// call after disconnected
	OnDisconnected func(connection *Connection)
}

type SeedConfig struct {
	Seed      string
	SecKey    string
	PublicKey string
	publicKey cipher.PubKey
	secKey    cipher.SecKey
}

func (sc *SeedConfig) parse() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("invalid seed config %#v", sc)
		}
	}()
	var key cipher.PubKey
	key, err = cipher.PubKeyFromHex(sc.PublicKey)
	if err != nil {
		return
	}
	sc.publicKey = key
	var secKey cipher.SecKey
	secKey, err = cipher.SecKeyFromHex(sc.SecKey)
	if err != nil {
		return
	}
	sc.secKey = secKey
	return
}

func NewSeedConfig() *SeedConfig {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return nil
	}
	seed, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil
	}
	pk, sk := cipher.GenerateDeterministicKeyPair([]byte(seed))
	sc := &SeedConfig{
		PublicKey: pk.Hex(),
		SecKey:    sk.Hex(),
		Seed:      seed,
		publicKey: pk,
		secKey:    sk,
	}
	return sc
}

func ReadSeedConfig(path string) (sc *SeedConfig, err error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	sc = &SeedConfig{}
	err = json.Unmarshal(fb, sc)
	if err == nil {
		err = sc.parse()
	}
	return
}

var readOrCreateMutex sync.Mutex

func ReadOrCreateSeedConfig(path string) (sc *SeedConfig, err error) {
	readOrCreateMutex.Lock()
	defer readOrCreateMutex.Unlock()
	sc, err = ReadSeedConfig(path)
	if err != nil {
		if os.IsNotExist(err) {
			sc = NewSeedConfig()
			err = WriteSeedConfig(sc, path)
			if err != nil {
				err = fmt.Errorf("failed to write seed config  %v", err)
				return
			}
		} else {
			err = fmt.Errorf("failed to read seed config %v", err)
			return
		}
	}
	return
}

func WriteSeedConfig(sc *SeedConfig, path string) (err error) {
	d, err := json.Marshal(sc)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, d, 0600)
	return
}
