package manager

import (
	"net/http"
	"time"

	"github.com/skycoin/skywire/pkg/cipher"
)

type Config struct {
	PK          cipher.PubKey
	SK          cipher.SecKey
	DBPath      string
	NamePattern string // regular expression for usernames (no check if empty). TODO
	PassPattern string // regular expression for passwords (no check of empty). TODO
	PassSaltLen int    // Salt Len for password verification data.
	Cookies     CookieConfig
}

func MakeConfig(dbPath string) Config {
	var c Config
	pk, sk := cipher.GenerateKeyPair()
	c.PK = pk
	c.SK = sk
	c.DBPath = dbPath
	c.Cookies.HashKey = cipher.RandByte(64)
	c.Cookies.BlockKey = cipher.RandByte(32)
	c.FillDefaults()
	return c
}

func (c *Config) FillDefaults() {
	c.NamePattern = `^(admin)$`
	c.PassPattern = `((?=.*\d)(?=.*[a-z])(?=.*[A-Z]).{6,20})`
	c.PassSaltLen = 16
	c.Cookies.FillDefaults()
}

type CookieConfig struct {
	HashKey  []byte // 32 or 64 bytes.
	BlockKey []byte // 16 (AES-128), 24 (AES-192), 32 (AES-256) bytes. (optional)

	ExpiresDuration time.Duration

	Path     string // optional
	Domain   string // optional
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

func (c *CookieConfig) FillDefaults() {
	c.Path = "/"
	c.ExpiresDuration = time.Hour * 12
	c.Secure = true
	c.HttpOnly = true
	c.SameSite = http.SameSiteDefaultMode
}
