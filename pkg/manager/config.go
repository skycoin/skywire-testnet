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

// Key allows a byte slice to be marshaled or unmarshaled from a hex string.
type Key []byte

// String implements fmt.Stringer
func (hk Key) String() string {
	return hex.EncodeToString(hk)
}

// MarshalText implements encoding.TextMarshaler
func (hk Key) MarshalText() ([]byte, error) {
	return []byte(hk.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (hk *Key) UnmarshalText(text []byte) error {
	*hk = make([]byte, hex.DecodedLen(len(text)))
	_, err := hex.Decode(*hk, text)
	return err
}

// Config configures the manager node.
type Config struct {
	PK         cipher.PubKey   `json:"public_key"`
	SK         cipher.SecKey   `json:"secret_key"`
	DBPath     string          `json:"db_path"`    // Path to store database file.
	Cookies    CookieConfig    `json:"cookies"`    // Configures cookies (for session management).
	Interfaces InterfaceConfig `json:"interfaces"` // Configures exposed interfaces.
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

// GenerateHomeConfig generates a config with default values and uses db from user's home folder.
func GenerateHomeConfig() Config {
	c := makeConfig()
	c.DBPath = filepath.Join(pathutil.HomeDir(), ".skycoin/skywire-manager/users.db")
	return c
}

// GenerateLocalConfig generates a config with default values and uses db from shared folder.
func GenerateLocalConfig() Config {
	c := makeConfig()
	c.DBPath = "/usr/local/skycoin/skywire-manager/users.db"
	return c
}

// FillDefaults fills the config with default values.
func (c *Config) FillDefaults() {
	c.Cookies.FillDefaults()
	c.Interfaces.FillDefaults()
}

// Parse parses the file in path, and decodes to the config.
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

// CookieConfig configures cookies used for manager.
type CookieConfig struct {
	HashKey  Key `json:"hash_key"`  // Signs the cookie: 32 or 64 bytes.
	BlockKey Key `json:"block_key"` // Encrypts the cookie: 16 (AES-128), 24 (AES-192), 32 (AES-256) bytes. (optional)

	ExpiresDuration time.Duration `json:"expires_duration"` // Used for determining the 'expires' value for cookies.

	Path     string        `json:"path"`   // optional
	Domain   string        `json:"domain"` // optional
	Secure   bool          `json:"secure"`
	HTTPOnly bool          `json:"http_only"`
	SameSite http.SameSite `json:"same_site"`
}

// FillDefaults fills config with default values.
func (c *CookieConfig) FillDefaults() {
	c.ExpiresDuration = time.Hour * 12
	c.Path = "/"
	c.Secure = true
	c.HTTPOnly = true
	c.SameSite = http.SameSiteDefaultMode
}

// InterfaceConfig configures the interfaces exposed by manager.
type InterfaceConfig struct {
	HTTPAddr string `json:"http_address"`
	RPCAddr  string `json:"rpc_addr"`
}

// FillDefaults fills config with default values.
func (c *InterfaceConfig) FillDefaults() {
	c.HTTPAddr = ":8080"
	c.RPCAddr = ":7080"
}
