package factory

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/go-bip39"
)

type ConnConfig struct {
	Reconnect     bool
	ReconnectWait time.Duration
	Creator       *MessengerFactory

	// generate seed, private key and public key for the connection
	// seed config file path
	SeedConfigPath string
	SeedConfig     *SeedConfig

	// context
	Context map[string]string

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
	sc := &SeedConfig{PublicKey: pk.Hex(), SecKey: sk.Hex(), Seed: seed}
	return sc
}

func ReadSeedConfig(path string) (sc *SeedConfig, err error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	sc = &SeedConfig{}
	err = json.Unmarshal(fb, sc)
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
