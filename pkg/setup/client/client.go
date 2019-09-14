package client

import (
	"context"
	"errors"
	"net"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/snet"
)

// Client interacts with setup nodes.
type Client interface {
	Dial(context.Context) (*snet.Conn, error)
	Serve() error
	IsTrusted(sPK cipher.PubKey) bool
}

type handlerFunc func(net.Conn) error

type client struct {
	Logger *logging.Logger

	network *snet.Network
	sl      *snet.Listener
	nodes   []cipher.PubKey
	handler handlerFunc
}

// New returns a new setup node client instance.
func New(network *snet.Network, sl *snet.Listener, nodes []cipher.PubKey, handler handlerFunc) Client {
	c := &client{
		Logger: logging.MustGetLogger("setup-client"),

		network: network,
		sl:      sl,
		nodes:   nodes,
		handler: handler,
	}

	return c
}

// TODO: use context
func (c *client) Dial(ctx context.Context) (*snet.Conn, error) {
	for _, sPK := range c.nodes {
		conn, err := c.network.Dial(snet.DmsgType, sPK, snet.SetupPort)
		if err != nil {
			c.Logger.WithError(err).Warnf("failed to dial to setup node: setupPK(%s)", sPK)
			continue
		}
		return conn, nil
	}
	return nil, errors.New("failed to dial to a setup node")
}

// ServeConnLoop initiates serving connections by route manager.
func (c *client) Serve() error {
	// Accept setup node request loop.
	for {
		if err := c.serveConn(); err != nil {
			return err
		}
	}
}

func (c *client) serveConn() error {
	conn, err := c.sl.AcceptConn()
	if err != nil {
		c.Logger.WithError(err).Warnf("stopped serving")
		return err
	}
	if !c.IsTrusted(conn.RemotePK()) {
		c.Logger.Warnf("closing conn from untrusted setup node: %v", conn.Close())
		return nil
	}
	go func() {
		c.Logger.Infof("handling setup request: setupPK(%s)", conn.RemotePK())
		if err := c.handler(conn); err != nil {
			c.Logger.WithError(err).Warnf("setup request failed: setupPK(%s)", conn.RemotePK())
		}
		c.Logger.Infof("successfully handled setup request: setupPK(%s)", conn.RemotePK())
	}()
	return nil
}

// SetupIsTrusted checks if setup node is trusted.
func (c *client) IsTrusted(sPK cipher.PubKey) bool {
	for _, pk := range c.nodes {
		if sPK == pk {
			return true
		}
	}
	return false
}
