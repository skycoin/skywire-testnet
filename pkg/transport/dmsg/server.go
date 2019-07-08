package dmsg

import (
	"net"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
)

type Server = dmsg.Server

func NewServer(pk cipher.PubKey, sk cipher.SecKey, addr string, l net.Listener, dc disc.APIClient) (*Server, error) {
	return dmsg.NewServer(pk, sk, addr, l, dc)
}
