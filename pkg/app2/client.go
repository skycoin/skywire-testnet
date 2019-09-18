package app2

import (
	"context"
	"net"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/netutil"
	"github.com/skycoin/skywire/pkg/routing"
)

// Client is used by skywire apps.
type Client struct {
	pk     cipher.PubKey
	pid    ProcID
	rpc    ServerRPCClient
	porter *netutil.Porter
}

// NewClient creates a new `Client`. The `Client` needs to be provided with:
// - log: Logger instance.
// - localPK: The local public key of the parent skywire visor.
// - pid: The procID assigned for the process that Client is being used by.
// - rpc: RPC client to communicate with the server.
func NewClient(localPK cipher.PubKey, pid ProcID, rpc ServerRPCClient, porter *netutil.Porter) *Client {
	return &Client{
		pk:     localPK,
		pid:    pid,
		rpc:    rpc,
		porter: porter,
	}
}

// Dial dials the remote node using `remote`.
func (c *Client) Dial(remote routing.Addr) (net.Conn, error) {
	localPort, free, err := c.porter.ReserveEphemeral(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	connID, err := c.rpc.Dial(remote)
	if err != nil {
		free()
		return nil, err
	}

	conn := &Conn{
		id:  connID,
		rpc: c.rpc,
		local: routing.Addr{
			PubKey: c.pk,
			Port:   routing.Port(localPort),
		},
		remote:        remote,
		freeLocalPort: free,
	}

	return conn, nil
}

// Listen listens on the specified `port` for the incoming connections.
func (c *Client) Listen(port routing.Port) (net.Listener, error) {
	ok, free := c.porter.Reserve(uint16(port), nil)
	if !ok {
		return nil, ErrPortAlreadyBound
	}

	local := routing.Addr{
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