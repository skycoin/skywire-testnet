package routerclient

import (
	"context"
	"net/rpc"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/snet"
)

const rpcName = "Gateway"

type Client struct {
	tr  *dmsg.Transport
	rpc *rpc.Client
}

func NewClient(ctx context.Context, dmsgC *dmsg.Client, pk cipher.PubKey) (*Client, error) {
	tr, err := dmsgC.Dial(ctx, pk, snet.AwaitSetupPort)
	if err != nil {
		return nil, err
	}

	client := &Client{
		tr:  tr,
		rpc: rpc.NewClient(tr.Conn),
	}
	return client, nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	if err := c.tr.Close(); err != nil {
		return err
	}

	if err := c.rpc.Close(); err != nil {
		return err
	}

	return nil
}

func (c *Client) AddEdgeRules(ctx context.Context, rules routing.EdgeRules) (bool, error) {
	var ok bool
	err := c.call(ctx, rpcName+".AddEdgeRules", rules, &ok)

	return ok, err
}

func (c *Client) AddIntermediaryRules(ctx context.Context, rules []routing.Rule) (bool, error) {
	var ok bool
	err := c.call(ctx, rpcName+".AddIntermediaryRules", rules, &ok)

	return ok, err
}

func (c *Client) ReserveIDs(ctx context.Context, n uint8) ([]routing.RouteID, error) {
	var routeIDs []routing.RouteID
	err := c.call(ctx, rpcName+".ReserveIDs", n, &routeIDs)

	return routeIDs, err
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
