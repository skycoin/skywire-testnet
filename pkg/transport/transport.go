// Package transport defines transport related entities and management
// operations.
package transport

import (
	"context"
	"time"

	"github.com/skycoin/skywire/pkg/cipher"
)

// Transport represents communication between two nodes via a single hop.
type Transport interface {

	// Read implements io.Reader
	Read(p []byte) (n int, err error)

	// Write implements io.Writer
	Write(p []byte) (n int, err error)

	// Close implements io.Closer
	Close() error

	// Edges returns sorted edges of transport
	Edges() [2]cipher.PubKey

	// SetDeadline functions the same as that from net.Conn
	// With a Transport, we don't have a distinction between write and read timeouts.
	SetDeadline(t time.Time) error

	// Type returns the string representation of the transport type.
	Type() string
}

// Factory generates Transports of a certain type.
type Factory interface {

	// Accept accepts a remotely-initiated Transport.
	Accept(ctx context.Context) (Transport, error)

	// Dial initiates a Transport with a remote node.
	Dial(ctx context.Context, remote cipher.PubKey) (Transport, error)

	// Close implements io.Closer
	Close() error

	// Local returns the local public key.
	Local() cipher.PubKey

	// Type returns the Transport type.
	Type() string
}
