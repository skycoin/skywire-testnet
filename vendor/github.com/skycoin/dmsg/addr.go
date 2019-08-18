package dmsg

import (
	"fmt"

	"github.com/skycoin/dmsg/cipher"
)

// Addr implements net.Addr for skywire addresses.
type Addr struct {
	PK   cipher.PubKey
	Port uint16
}

// Network returns "dmsg"
func (Addr) Network() string {
	return Type
}

// String returns public key and port of node split by colon.
func (a Addr) String() string {
	if a.Port == 0 {
		return fmt.Sprintf("%s:~", a.PK)
	}
	return fmt.Sprintf("%s:%d", a.PK, a.Port)
}
