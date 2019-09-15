package app2

import (
	"net/rpc"

	"github.com/skycoin/skywire/pkg/routing"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
)

// Client is used by skywire apps.
type Client struct {
	PK     cipher.PubKey
	pid    ProcID
	rpc    ServerRPCClient
	logger *logging.Logger
}

// NewClient creates a new Client. The Client needs to be provided with:
// - localPK: The local public key of the parent skywire visor.
// - pid: The procID assigned for the process that Client is being used by.
// - sockAddr: The socket address to connect to Server.
func NewClient(localPK cipher.PubKey, pid ProcID, rpc *rpc.Client, l *logging.Logger) *Client {
	return &Client{
		PK:     localPK,
		pid:    pid,
		rpc:    newServerRPCClient(rpc),
		logger: l,
	}
}

func (c *Client) Dial(remote routing.Addr) (*Conn, error) {
	connID, err := c.rpc.Dial(remote)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		id:  connID,
		rpc: c.rpc,
		// TODO: port?
		local: routing.Addr{
			PubKey: c.PK,
		},
		remote: remote,
	}

	return conn, nil
}

func (c *Client) Listen(port routing.Port) (*Listener, error) {
	local := routing.Addr{
		PubKey: c.PK,
		Port:   port,
	}

	lisID, err := c.rpc.Listen(local)
	if err != nil {
		return nil, err
	}

	listener := &Listener{
		id:   lisID,
		rpc:  c.rpc,
		addr: local,
	}

	return listener, nil
}
