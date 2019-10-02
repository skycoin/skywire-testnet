package app

import "github.com/skycoin/skywire/pkg/routing"

// Packet represents message exchanged between App and Node.
type Packet struct {
	Desc    routing.RouteDescriptor `json:"desc"`
	Payload []byte                  `json:"payload"`
}
