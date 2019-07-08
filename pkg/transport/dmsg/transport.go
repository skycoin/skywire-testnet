package dmsg

import (
	"time"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/transport"
)

// Transport is a wrapper type for "github.com/skycoin/dmsg".Transport
type Transport struct {
	*dmsg.Transport
}

// NewTransport creates a new Transport.
func NewTransport(tp *dmsg.Transport) *Transport {
	return &Transport{Transport: tp}
}

// Read is a wrapper for "github.com/skycoin/dmsg".(*Transport).Read
func (tp *Transport) Read(p []byte) (n int, err error) {
	return tp.Transport.Read(p)
}

// Write is a wrapper for "github.com/skycoin/dmsg".(*Transport).Write
func (tp *Transport) Write(p []byte) (n int, err error) {
	return tp.Transport.Write(p)
}

// Close is a wrapper for "github.com/skycoin/dmsg".(*Transport).Close
func (tp *Transport) Close() error {
	return tp.Transport.Close()
}

// Edges returns sorted edges of transport
func (tp *Transport) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(tp.LocalPK(), tp.RemotePK())
}

// SetDeadline is a wrapper for "github.com/skycoin/dmsg".(*Transport).SetDeadline
func (tp *Transport) SetDeadline(t time.Time) error {
	return tp.Transport.SetDeadline(t)
}

// Type is a wrapper for "github.com/skycoin/dmsg".(*Transport).Type
func (tp *Transport) Type() string {
	return tp.Transport.Type()
}
