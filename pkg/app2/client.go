package app2

import (
	"net"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app2/network"
	"github.com/skycoin/skywire/pkg/routing"
)

// Client is used by skywire apps.
type Client struct {
	log *logging.Logger
	pk  cipher.PubKey
	pid ProcID
	rpc RPCClient
	lm  *idManager // contains listeners associated with their IDs
	cm  *idManager // contains connections associated with their IDs
}

// NewClient creates a new `Client`. The `Client` needs to be provided with:
// - log: logger instance
// - localPK: The local public key of the parent skywire visor.
// - pid: The procID assigned for the process that Client is being used by.
// - rpc: RPC client to communicate with the server.
func NewClient(log *logging.Logger, localPK cipher.PubKey, pid ProcID, rpc RPCClient) *Client {
	return &Client{
		log: log,
		pk:  localPK,
		pid: pid,
		rpc: rpc,
		lm:  newIDManager(),
		cm:  newIDManager(),
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

	free, err := c.cm.add(connID, conn)
	if err != nil {
		if err := conn.Close(); err != nil {
			c.log.WithError(err).Error("error closing conn")
		}

		return nil, err
	}

	conn.freeConn = free

	return conn, nil
}

// Listen listens on the specified `port` for the incoming connections.
func (c *Client) Listen(n network.Type, port routing.Port) (net.Listener, error) {
	local := network.Addr{
		Net:    n,
		PubKey: c.pk,
		Port:   port,
	}

	lisID, err := c.rpc.Listen(local)
	if err != nil {
		return nil, err
	}

	listener := &Listener{
		log:  c.log,
		id:   lisID,
		rpc:  c.rpc,
		addr: local,
		cm:   newIDManager(),
	}

	freeLis, err := c.lm.add(lisID, listener)
	if err != nil {
		if err := listener.Close(); err != nil {
			c.log.WithError(err).Error("error closing listener")
		}

		return nil, err
	}

	listener.freeLis = freeLis

	return listener, nil
}

// Close closes client/server communication entirely. It closes all open
// listeners and connections.
func (c *Client) Close() {
	var listeners []net.Listener
	c.lm.doRange(func(_ uint16, v interface{}) bool {
		lis, err := assertListener(v)
		if err != nil {
			c.log.Error(err)
			return true
		}

		listeners = append(listeners, lis)
		return true
	})

	var conns []net.Conn
	c.cm.doRange(func(_ uint16, v interface{}) bool {
		conn, err := assertConn(v)
		if err != nil {
			c.log.Error(err)
			return true
		}

		conns = append(conns, conn)
		return true
	})

	for _, lis := range listeners {
		if err := lis.Close(); err != nil {
			c.log.WithError(err).Error("error closing listener")
		}
	}

	for _, conn := range conns {
		if err := conn.Close(); err != nil {
			c.log.WithError(err).Error("error closing conn")
		}
	}
}
