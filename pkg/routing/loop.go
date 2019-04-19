package routing

import (
	"fmt"
	"time"

	"github.com/skycoin/skywire/pkg/cipher"
)

// Loop defines a loop over a pair of routes.
type Loop struct {
	LocalPort    uint16 
	RemotePort   uint16
	Forward      Route
	Reverse      Route
	Expiry       time.Time
	NoiseMessage []byte  
}

// Initiator returns initiator of the Loop.
func (l *Loop) Initiator() cipher.PubKey {
	if len(l.Forward) == 0 {
		panic("empty forward route")
	}

	return l.Forward[0].From
}

// Responder returns responder of the Loop.
func (l *Loop) Responder() cipher.PubKey {
	if len(l.Reverse) == 0 {
		panic("empty reverse route")
	}

	return l.Reverse[0].From
}

func (l *Loop) String() string {
	return fmt.Sprintf("lport: %d. rport: %d. routes: %s/%s. expire at %s",
		l.LocalPort, l.RemotePort, l.Forward, l.Reverse, l.Expiry)
}
