package dmsg

import (
	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/transport"
)

const (
	// PurposeData means that transport is used for data transmission.
	PurposeData = dmsg.PurposeData
	// PurposeSetup means that transport is used for setup nodes.
	PurposeSetup = dmsg.PurposeSetup
	// PurposeTest means that transport is used for tests.
	PurposeTest = dmsg.PurposeTest
)

// Transport is a wrapper type for "github.com/skycoin/dmsg".Transport
type Transport struct {
	*dmsg.Transport
}

// NewTransport creates a new Transport.
func NewTransport(tp *dmsg.Transport) *Transport {
	return &Transport{Transport: tp}
}

// Edges returns sorted edges of transport
func (tp *Transport) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(tp.LocalPK(), tp.RemotePK())
}
