package network

import (
	"errors"
	"fmt"
	"net"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/routing"
)

var (
	errUnknownAddrType = errors.New("addr type is unknown")
)

// Addr implements net.Addr for network addresses.
type Addr struct {
	Net    Type
	PubKey cipher.PubKey
	Port   routing.Port
}

// Network returns network type.
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

// WrapAddr asserts type of the passed `net.Addr` and converts it
// to `Addr` if possible.
func WrapAddr(addr net.Addr) (Addr, error) {
	switch a := addr.(type) {
	case dmsg.Addr:
		return Addr{
			Net:    TypeDMSG,
			PubKey: a.PK,
			Port:   routing.Port(a.Port),
		}, nil
	default:
		return Addr{}, errUnknownAddrType
	}
}
