package dmsg

import (
	"context"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/transport"
)

const (
	// Type is a wrapper type for "github.com/skycoin/dmsg".Type
	Type = dmsg.Type
)

// Client is a wrapper type for "github.com/skycoin/dmsg".Client
type Client struct {
	*dmsg.Client
}

// ClientOption is a wrapper type for "github.com/skycoin/dmsg".ClientOption
type ClientOption = dmsg.ClientOption

// NewClient is a wrapper type for "github.com/skycoin/dmsg".NewClient
func NewClient(pk cipher.PubKey, sk cipher.SecKey, dc disc.APIClient, opts ...ClientOption) *Client {
	return &Client{
		Client: dmsg.NewClient(pk, sk, dc, opts...),
	}
}

// Accept is a wrapper type for "github.com/skycoin/dmsg".Accept
func (c *Client) Accept(ctx context.Context) (transport.Transport, error) {
	tp, err := c.Client.Accept(ctx)
	if err != nil {
		return nil, err
	}

	return NewTransport(tp), nil
}

// Dial is a wrapper type for "github.com/skycoin/dmsg".Dial
func (c *Client) Dial(ctx context.Context, remote cipher.PubKey) (transport.Transport, error) {
	tp, err := c.Client.Dial(ctx, remote)
	if err != nil {
		return nil, err
	}
	return NewTransport(tp), nil
}

// Close is a wrapper type for "github.com/skycoin/dmsg".Close
func (c *Client) Close() error {
	return c.Client.Close()
}

// Local is a wrapper type for "github.com/skycoin/dmsg".Local
func (c *Client) Local() cipher.PubKey {
	return c.Client.Local()
}

// Type is a wrapper type for "github.com/skycoin/dmsg".Type
func (c *Client) Type() string {
	return c.Client.Type()
}

// SetLogger is a wrapper type for "github.com/skycoin/dmsg".SetLogger
func SetLogger(log *logging.Logger) ClientOption {
	return dmsg.SetLogger(log)
}
