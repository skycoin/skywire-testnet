// Package transport defines transport related entities and management
// operations.
package transport

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
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

	// Edges returns sorted edges of transport
	Edges() [2]cipher.PubKey

	// SetDeadline functions the same as that from net.Conn
	// With a Transport, we don't have a distinction between write and read timeouts.
	SetDeadline(t time.Time) error

	// Type returns the string representation of the transport type.
	Type() string

	// Purpose returns the string representation of the transport purpose.
	Purpose() string
}

// Factory generates Transports of a certain type.
type Factory interface {

	// Accept accepts a remotely-initiated Transport.
	Accept(ctx context.Context) (Transport, error)

	// Dial initiates a Transport with a remote node.
	Dial(ctx context.Context, remote cipher.PubKey, purpose string) (Transport, error)

	// Close implements io.Closer
	Close() error

	// Local returns the local public key.
	Local() cipher.PubKey

	// Type returns the Transport type.
	Type() string
}

// MakeTransportID generates uuid.UUID from pair of keys + type + purpose + public
// Generated uuid is:
// - always the same for a given pair
// - GenTransportUUID(keyA,keyB) == GenTransportUUID(keyB, keyA)
func MakeTransportID(keyA, keyB cipher.PubKey, tpType, purpose string, public bool) uuid.UUID {
	keys := SortPubKeys(keyA, keyB)
	if public {
		return uuid.NewSHA1(uuid.UUID{},
			append(append(append(append(keys[0][:], keys[1][:]...), []byte(tpType)...), []byte(purpose)...), 1))
	}
	return uuid.NewSHA1(uuid.UUID{},
		append(append(append(append(keys[0][:], keys[1][:]...), []byte(tpType)...), []byte(purpose)...), 0))
}

// SortPubKeys sorts keys so that least-significant comes first
func SortPubKeys(keyA, keyB cipher.PubKey) [2]cipher.PubKey {
	for i := 0; i < 33; i++ {
		if keyA[i] != keyB[i] {
			if keyA[i] < keyB[i] {
				return [2]cipher.PubKey{keyA, keyB}
			}
			return [2]cipher.PubKey{keyB, keyA}
		}
	}
	return [2]cipher.PubKey{keyA, keyB}
}

// SortEdges sorts edges so that list-significant comes firs
func SortEdges(edges [2]cipher.PubKey) [2]cipher.PubKey {
	return SortPubKeys(edges[0], edges[1])
}
