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

	TransportType    string `json:"transport_type"`
	PubKeysFile      string `json:"pubkeys_file"`
	TCPTransportAddr string `json:"tcptransport_addr"`

	LogLevel string `json:"log_level"`
}
