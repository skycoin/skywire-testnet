package app

import (
	"fmt"

	"github.com/skycoin/skywire/pkg/cipher"
)

// LoopAddr stores addressing parameters of a loop packets.
type LoopAddr struct {
	Port   uint16 `json:"port"`
	Remote Addr   `json:"remote"`
}

func (l *LoopAddr) String() string {
	return fmt.Sprintf(":%d <-> %s:%d", l.Port, l.Remote.PubKey, l.Remote.Port)
}

// Packet represents message exchanged between App and Node.
type Packet struct {
	Addr    *LoopAddr `json:"addr"`
	Payload []byte    `json:"payload"`
}

// Addr implements net.Addr for App connections.
type Addr struct {
	PubKey cipher.PubKey `json:"pk"`
	Port   uint16        `json:"port"`
}

// Network returns custom skywire Network type.
func (addr *Addr) Network() string {
	return "skywire"
}

func (addr *Addr) String() string {
	return fmt.Sprintf("%s:%d", addr.PubKey, addr.Port)
}
