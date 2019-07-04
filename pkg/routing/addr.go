package routing

import "github.com/skycoin/dmsg/cipher"

// Addr represents a network address combining public key and port
type Addr struct {
	PubKey cipher.PubKey `json:"pk"`
	Port   uint16        `json:"port"`
}
