// Package routing defines routing related entities and management
// operations.
package routing

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
)

// Hop defines a route hop between 2 nodes.
type Hop struct {
	TpID uuid.UUID
	From cipher.PubKey
	To   cipher.PubKey
}

func (h Hop) String() string {
	return fmt.Sprintf("%s -> %s @ %s", h.From, h.To, h.TpID)
}

// Route is a succession of transport entries that denotes a path from source node to destination node
type Route struct {
	Desc      RouteDescriptor `json:"desc"`
	Hops      []Hop           `json:"hops"`
	KeepAlive time.Duration   `json:"keep_alive"`
}

func (r Route) String() string {
	res := "\n"
	for _, hop := range r.Hops {
		res += fmt.Sprintf("\t%s\n", hop)
	}

	return res
}
