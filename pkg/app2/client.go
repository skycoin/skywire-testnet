package app2

import (
	"net"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/netutil"

	"github.com/skycoin/skywire/pkg/app2/network"
	"github.com/skycoin/skywire/pkg/routing"
)

// Client is used by skywire apps.
type Client struct {
	pk     cipher.PubKey
	pid    ProcID
	rpc    RPCClient
	porter *netutil.Porter
}

// NewClient creates a new `Client`. The `Client` needs to be provided with:
// - localPK: The local public key of the parent skywire visor.
// - pid: The procID assigned for the process that Client is being used by.
// - rpc: RPC client to communicate with the server.
func NewClient(localPK cipher.PubKey, pid ProcID, rpc RPCClient) *Client {
	return &Client{
		pk:     localPK,
		pid:    pid,
		rpc:    rpc,
		porter: netutil.NewPorter(netutil.PorterMinEphemeral),
	}
}

// Dial dials the remote node using `remote`.
func (c *Client) Dial(remote network.Addr) (net.Conn, error) {
	connID, assignedPort, err := c.rpc.Dial(remote)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		id:  connID,
		rpc: c.rpc,
		local: network.Addr{
			Net:    remote.Net,
			PubKey: c.pk,
			Port:   assignedPort,
		},
		remote: remote,
	}

	return conn, nil
}

// Listen listens on the specified `port` for the incoming connections.
func (c *Client) Listen(n network.Type, port routing.Port) (net.Listener, error) {
	ok, free := c.porter.Reserve(uint16(port), nil)
	if !ok {
		return nil, ErrPortAlreadyBound
	}

	local := network.Addr{
		Net:    n,
		PubKey: c.pk,
		Port:   port,
	}

	lisID, err := c.rpc.Listen(local)
	if err != nil {
		free()
		return nil, err
	}

	listener := &Listener{
		id:       lisID,
		rpc:      c.rpc,
		addr:     local,
		freePort: free,
	}

	return listener, nil
}
