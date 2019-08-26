package routing

import (
	"fmt"
	"time"

	"github.com/skycoin/dmsg/cipher"
)

// AddressPair defines a loop over a pair of addresses.
type AddressPair struct {
	Local  Addr
	Remote Addr
}

// TODO: discuss if we should add local PK to the output
func (apd AddressPair) String() string {
	return fmt.Sprintf("%s:%d <-> %s:%d", apd.Local.PubKey, apd.Local.Port, apd.Remote.PubKey, apd.Remote.Port)
}

// AddressPairDescriptor defines a loop over a pair of routes.
type AddressPairDescriptor struct {
	Loop    AddressPair
	Forward Route
	Reverse Route
	Expiry  time.Time
}

// Initiator returns initiator of the Loop.
func (apd AddressPairDescriptor) Initiator() cipher.PubKey {
	if len(apd.Forward) == 0 {
		panic("empty forward route")
	}

	return apd.Forward[0].From
}

// Responder returns responder of the Loop.
func (apd AddressPairDescriptor) Responder() cipher.PubKey {
	if len(apd.Reverse) == 0 {
		panic("empty reverse route")
	}

	return apd.Reverse[0].From
}

func (apd AddressPairDescriptor) String() string {
	return fmt.Sprintf("lport: %d. rport: %d. routes: %s/%s. expire at %s",
		apd.Loop.Local.Port, apd.Loop.Remote.Port, apd.Forward, apd.Reverse, apd.Expiry)
}

// AddressPairData stores loop confirmation request data.
type AddressPairData struct {
	Loop    AddressPair `json:"loop"`
	RouteID RouteID     `json:"resp-rid,omitempty"`
}
