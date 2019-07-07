package routing

import (
	"fmt"

	"github.com/skycoin/dmsg/cipher"
)

// Port is a network port number
type Port uint16

const networkType = "skywire"

// Addr represents a network address combining public key and port.
// Implements net.Addr
type Addr struct {
	PubKey cipher.PubKey `json:"pk"`
	Port   Port          `json:"port"`
}

// Network returns type of `a`'s network
func (a *Addr) Network() string {
	return networkType
}

func (a *Addr) String() string {
	return fmt.Sprintf("%s:%d", a.PubKey, a.Port)
}
