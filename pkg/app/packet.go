package app

import "github.com/skycoin/skywire/pkg/routing"

// Packet represents message exchanged between App and Node.
type Packet struct {
	Addr    *routing.Loop `json:"addr"`
	Payload []byte        `json:"payload"`
}
