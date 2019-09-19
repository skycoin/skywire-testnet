// Package routing defines routing related entities and management
// operations.
package routing

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/SkycoinProject/dmsg/cipher"
)

// Hop defines a route hop between 2 nodes.
type Hop struct {
	From      cipher.PubKey
	To        cipher.PubKey
	Transport uuid.UUID
}

func (h Hop) String() string {
	return fmt.Sprintf("%s -> %s @ %s", h.From, h.To, h.Transport)
}

// Route is a succession of transport entries that denotes a path from source node to destination node
type Route []*Hop

func (r Route) String() string {
	res := "\n"
	for _, hop := range r {
		res += fmt.Sprintf("\t%s\n", hop)
	}

	return res
}
