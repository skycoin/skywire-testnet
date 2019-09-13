package routing

import (
	"fmt"
	"time"

	"github.com/skycoin/dmsg/cipher"
)

// Loop defines a loop over a pair of addresses.
type Loop struct {
	Local  Addr
	Remote Addr
}

// TODO: discuss if we should add local PK to the output
func (l Loop) String() string {
	return fmt.Sprintf("%s:%d <-> %s:%d", l.Local.PubKey, l.Local.Port, l.Remote.PubKey, l.Remote.Port)
}

// LoopDescriptor defines a loop over a pair of routes.
type LoopDescriptor struct {
	Loop      Loop
	Forward   Path
	Reverse   Path
	KeepAlive time.Duration
}

// Initiator returns initiator of the Loop.
func (l LoopDescriptor) Initiator() cipher.PubKey {
	if len(l.Forward) == 0 {
		panic("empty forward route")
	}

	return l.Forward[0].From
}

// Responder returns responder of the Loop.
func (l LoopDescriptor) Responder() cipher.PubKey {
	if len(l.Reverse) == 0 {
		panic("empty reverse route")
	}

	return l.Reverse[0].From
}

func (l LoopDescriptor) String() string {
	return fmt.Sprintf("lport: %d. rport: %d. routes: %s/%s. keep-alive timeout %s",
		l.Loop.Local.Port, l.Loop.Remote.Port, l.Forward, l.Reverse, l.KeepAlive)
}

// LoopData stores loop confirmation request data.
type LoopData struct {
	Loop    Loop    `json:"loop"`
	RouteID RouteID `json:"resp-rid,omitempty"`
}
