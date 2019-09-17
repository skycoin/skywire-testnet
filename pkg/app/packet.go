package app

import "github.com/SkycoinProject/skywire/pkg/routing"

// Packet represents message exchanged between App and Node.
type Packet struct {
	Loop    routing.Loop `json:"loop"`
	Payload []byte       `json:"payload"`
}
