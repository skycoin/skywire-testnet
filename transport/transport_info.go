package transport

//	for cli output

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/skywire/src/messages"
)

type TransportInfo struct {
	TransportId messages.TransportId
	Status      uint8
	NodeFrom    cipher.PubKey
	NodeTo      cipher.PubKey
}
