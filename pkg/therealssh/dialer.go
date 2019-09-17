package therealssh

import (
	"net"

	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
)

// dialer dials to a remote node.
type dialer interface {
	Dial(raddr routing.Addr) (net.Conn, error)
}
