// Package transport defines transport related entities and management
// operations.
package transport

import (
	"crypto/sha256"
	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"math/big"
)

var log = logging.MustGetLogger("transport")

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
