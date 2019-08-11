package dmsg

import (
	"context"
	"net"
	"time"

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

// Config configures dmsg
type Config struct {
	PubKey     cipher.PubKey
	SecKey     cipher.SecKey
	Discovery  disc.APIClient
	Retries    int
	RetryDelay time.Duration
}

// Server is an alias for dmsg.Server.
type Server = dmsg.Server

// NewServer is an alias for dmsg.NewServer.
func NewServer(pk cipher.PubKey, sk cipher.SecKey, addr string, l net.Listener, dc disc.APIClient) (*Server, error) {
	return dmsg.NewServer(pk, sk, addr, l, dc)
}

// ClientOption is a wrapper type for "github.com/skycoin/dmsg".ClientOption
type ClientOption = dmsg.ClientOption

// Client is a wrapper type for "github.com/skycoin/dmsg".Client
type Client struct {
	*dmsg.Client
}

// NewClient is a wrapper type for "github.com/skycoin/dmsg".NewClient
func NewClient(pk cipher.PubKey, sk cipher.SecKey, dc disc.APIClient, opts ...ClientOption) *Client {
	return &Client{
		Client: dmsg.NewClient(pk, sk, dc, opts...),
	}
}

// Accept is a wrapper type for "github.com/skycoin/dmsg".Accept
func (c *Client) Accept(ctx context.Context) (transport.Transport, error) {
	return c.Client.Accept(ctx)
}

// Dial is a wrapper type for "github.com/skycoin/dmsg".Dial
func (c *Client) Dial(ctx context.Context, remote cipher.PubKey) (transport.Transport, error) {
	return c.Client.Dial(ctx, remote)
}

// SetLogger is a wrapper type for "github.com/skycoin/dmsg".SetLogger
func SetLogger(log *logging.Logger) ClientOption {
	return dmsg.SetLogger(log)
}
