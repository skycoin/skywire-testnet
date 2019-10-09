package setupclient

import (
	"context"
	"errors"
	"net/rpc"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/snet"
)

const rpcName = "RPCGateway"

// Client is an RPC client for setup node.
type Client struct {
	log        *logging.Logger
	n          *snet.Network
	setupNodes []cipher.PubKey
	conn       *snet.Conn
	rpc        *rpc.Client
}

// NewClient creates a new Client.
func NewClient(ctx context.Context, log *logging.Logger, n *snet.Network, setupNodes []cipher.PubKey) (*Client, error) {
	client := &Client{
		log:        log,
		n:          n,
		setupNodes: setupNodes,
	}

	conn, err := client.dial(ctx)
	if err != nil {
		return nil, err
	}
	client.conn = conn

	client.rpc = rpc.NewClient(conn)

	return client, nil
}

func (c *Client) dial(ctx context.Context) (*snet.Conn, error) {
	for _, sPK := range c.setupNodes {
		conn, err := c.n.Dial(ctx, snet.DmsgType, sPK, snet.SetupPort)
		if err != nil {
			c.log.WithError(err).Warnf("failed to dial to setup node: setupPK(%s)", sPK)
			continue
		}
		return conn, nil
	}
	return nil, errors.New("failed to dial to a setup node")
}

// Close closes a Client.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	if err := c.rpc.Close(); err != nil {
		return err
	}

	if err := c.conn.Close(); err != nil {
		return err
	}

	return nil
}

// DialRouteGroup generates rules for routes from a visor and sends them to visors.
func (c *Client) DialRouteGroup(ctx context.Context, req routing.BidirectionalRoute) (routing.EdgeRules, error) {
	var resp routing.EdgeRules
	err := c.call(ctx, rpcName+".DialRouteGroup", req, &resp)
	return resp, err
}

func (c *Client) call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	call := c.rpc.Go(serviceMethod, args, reply, nil)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-call.Done:
		return call.Error
	}
}
