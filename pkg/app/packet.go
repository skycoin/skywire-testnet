package app

import (
	"fmt"

	"github.com/skycoin/skywire/pkg/routing"
)

// LoopAddr stores addressing parameters of a loop packets.
type LoopAddr struct {
	Port   uint16       `json:"port"`
	Remote routing.Addr `json:"remote"`
}

func (l *LoopAddr) String() string {
	return fmt.Sprintf(":%d <-> %s:%d", l.Port, l.Remote.PubKey, l.Remote.Port)
}

// Packet represents message exchanged between App and Node.
type Packet struct {
	Addr    *LoopAddr `json:"addr"`
	Payload []byte    `json:"payload"`
}
