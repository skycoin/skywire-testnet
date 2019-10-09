// Package routing defines routing related entities and management
// operations.
package routing

import (
	"bytes"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
)

// PathEdges are the edge nodes of a path
type PathEdges [2]cipher.PubKey

// MarshalText implements encoding.TextMarshaler
func (p PathEdges) MarshalText() ([]byte, error) {
	b1, err := p[0].MarshalText()
	if err != nil {
		return nil, err
	}
	b2, err := p[1].MarshalText()
	if err != nil {
		return nil, err
	}
	res := bytes.NewBuffer(b1)
	res.WriteString(":") // nolint
	res.Write(b2)        // nolint
	return res.Bytes(), nil
}

// UnmarshalText implements json.Unmarshaler
func (p *PathEdges) UnmarshalText(b []byte) error {
	err := p[0].UnmarshalText(b[:66])
	if err != nil {
		return err
	}
	err = p[1].UnmarshalText(b[67:])
	if err != nil {
		return err
	}
	return nil
}

// Hop defines a route hop between 2 nodes.
type Hop struct {
	TpID uuid.UUID
	From cipher.PubKey
	To   cipher.PubKey
}

// Path is a list of hops between nodes (transports), and indicates a route between the edges
type Path []Hop

func (h Hop) String() string {
	return fmt.Sprintf("%s -> %s @ %s", h.From, h.To, h.TpID)
}

// Route is a succession of transport entries that denotes a path from source node to destination node
type Route struct {
	Desc      RouteDescriptor `json:"desc"`
	Path      Path            `json:"path"`
	KeepAlive time.Duration   `json:"keep_alive"`
}

func (r Route) String() string {
	res := fmt.Sprintf("[KeepAlive: %s] %s\n", r.KeepAlive, r.Desc.String())
	for _, hop := range r.Path {
		res += fmt.Sprintf("\t%s\n", hop)
	}

	return res
}

// BidirectionalRoute is a Route with both forward and reverse Paths.
type BidirectionalRoute struct {
	Desc      RouteDescriptor
	KeepAlive time.Duration
	Forward   Path
	Reverse   Path
}

// EdgeRules represents edge forward and reverse rules. Edge rules are forward and consume rules.
type EdgeRules struct {
	Desc    RouteDescriptor
	Forward Rule
	Reverse Rule
}
