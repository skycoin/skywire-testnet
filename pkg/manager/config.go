package manager

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/skycoin/skywire/internal/pathutil"

	"github.com/skycoin/skywire/pkg/cipher"
)

type Key []byte

func (hk Key) String() string {
	return hex.EncodeToString(hk)
}

func (hk Key) MarshalText() ([]byte, error) {
	return []byte(hk.String()), nil
}

func (hk *Key) UnmarshalText(text []byte) error {
	*hk = make([]byte, hex.DecodedLen(len(text)))
	_, err := hex.Decode(*hk, text)
	return err
}

type Config struct {
	PK          cipher.PubKey   `json:"public_key"`
	SK          cipher.SecKey   `json:"secret_key"`
	DBPath      string          `json:"db_path"`
	NameRegexp  string          `json:"username_regexp"`   // regular expression for usernames (no check if empty). TODO
	PassRegexp  string          `json:"password_regexp"`   // regular expression for passwords (no check of empty). TODO
	PassSaltLen int             `json:"password_salt_len"` // Salt Len for password verification data.
	Cookies     CookieConfig    `json:"cookies"`
	Interfaces  InterfaceConfig `json:"interfaces"`
}

func makeConfig() Config {
	var c Config
	pk, sk := cipher.GenerateKeyPair()
	c.PK = pk
	c.SK = sk
	c.Cookies.HashKey = cipher.RandByte(64)
	c.Cookies.BlockKey = cipher.RandByte(32)
	c.FillDefaults()
	return c
}

func GenerateHomeConfig() Config {
	c := makeConfig()
	c.DBPath = filepath.Join(pathutil.HomeDir(), ".skycoin/skywire-manager/users.db")
	return c
}

func GenerateLocalConfig() Config {
	c := makeConfig()
	c.DBPath = "/usr/local/skycoin/skywire-manager/users.db"
	return c
}

func (c *Config) FillDefaults() {
	c.NameRegexp = `^(admin)$`
	c.PassRegexp = `((?=.*\d)(?=.*[a-z])(?=.*[A-Z]).{6,20})`
	c.PassSaltLen = 16
	c.Cookies.FillDefaults()
	c.Interfaces.FillDefaults()
}

func (c *Config) Parse(path string) error {
	var err error
	if path, err = filepath.Abs(path); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { catch(f.Close()) }()
	return json.NewDecoder(f).Decode(c)
}

type CookieConfig struct {
	HashKey  Key `json:"hash_key"`  // 32 or 64 bytes.
	BlockKey Key `json:"block_key"` // 16 (AES-128), 24 (AES-192), 32 (AES-256) bytes. (optional)

	ExpiresDuration time.Duration `json:"expires_duration"`

	Path     string        `json:"path"`   // optional
	Domain   string        `json:"domain"` // optional
	Secure   bool          `json:"secure"`
	HttpOnly bool          `json:"http_only"`
	SameSite http.SameSite `json:"same_site"`
}

func (c *CookieConfig) FillDefaults() {
	c.Path = "/"
	c.ExpiresDuration = time.Hour * 12
	c.Secure = true
	c.HttpOnly = true
	c.SameSite = http.SameSiteDefaultMode
}

type InterfaceConfig struct {
	HTTPAddr string `json:"http_address"`
	RPCAddr  string `json:"rpc_addr"`
}

func (c *InterfaceConfig) FillDefaults() {
	c.HTTPAddr = ":8080"
	c.RPCAddr = ":7080"
}
