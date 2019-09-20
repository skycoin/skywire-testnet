package network

import (
	"fmt"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
)

// Addr implements net.Addr for network addresses.
type Addr struct {
	Net    Type
	PubKey cipher.PubKey
	Port   routing.Port
}

// Network returns "dmsg"
func (a Addr) Network() string {
	return string(a.Net)
}

// String returns public key and port of node split by colon.
func (a Addr) String() string {
	if a.Port == 0 {
		return fmt.Sprintf("%s:~", a.PubKey)
	}
	return fmt.Sprintf("%s:%d", a.PubKey, a.Port)
}
