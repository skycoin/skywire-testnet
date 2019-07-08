package setup

import (
	"github.com/skycoin/dmsg/cipher"
)

// Config defines configuration parameters for setup Node.
type Config struct {
	PubKey cipher.PubKey `json:"public_key"`
	SecKey cipher.SecKey `json:"secret_key"`

	Messaging struct {
		Discovery   string `json:"discovery"`
		ServerCount int    `json:"server_count"`
	}

	TransportDiscovery string `json:"transport_discovery"`

	LogLevel string `json:"log_level"`
}
