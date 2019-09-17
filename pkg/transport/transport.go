// Package transport defines transport related entities and management
// operations.
package transport

import (
	"context"
	"crypto/sha256"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/skycoin/src/util/logging"
)

var log = logging.MustGetLogger("transport")

// Transport represents communication between two nodes via a single hop.
type Transport interface {

	// Read implements io.Reader
	Read(p []byte) (n int, err error)

	// Write implements io.Writer
	Write(p []byte) (n int, err error)

	// Close implements io.Closer
	Close() error

	// LocalPK returns local public key of transport
	LocalPK() cipher.PubKey

	// RemotePK returns remote public key of transport
	RemotePK() cipher.PubKey

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

// MakeTransportID generates uuid.UUID from pair of keys + type + public
// Generated uuid is:
// - always the same for a given pair
// - GenTransportUUID(keyA,keyB) == GenTransportUUID(keyB, keyA)
func MakeTransportID(keyA, keyB cipher.PubKey, tpType string) uuid.UUID {
	keys := SortEdges(keyA, keyB)
	b := make([]byte, 33*2+len(tpType))
	i := 0
	i += copy(b[i:], keys[0][:])
	i += copy(b[i:], keys[1][:])
	copy(b[i:], tpType)
	return uuid.NewHash(sha256.New(), uuid.UUID{}, b, 0)
}

// SortEdges sorts keys so that least-significant comes first
func SortEdges(keyA, keyB cipher.PubKey) [2]cipher.PubKey {
	var a, b big.Int
	if a.SetBytes(keyA[:]).Cmp(b.SetBytes(keyB[:])) < 0 {
		return [2]cipher.PubKey{keyA, keyB}
	}
	return [2]cipher.PubKey{keyB, keyA}
}
