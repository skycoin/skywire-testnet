// Package routing defines routing related entities and management
// operations.
package routing

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
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

// PathEdges are the edge nodes of a path
type PathEdges [2]cipher.PubKey

// PathEdgesText is used internally for marshaling and unmarshaling of PathEdges
type PathEdgesText struct {
	Edge1 cipher.PubKey `json:"edge_1"`
	Edge2 cipher.PubKey `json:"edge_2"`
}

// MarshalText implements encoding.TextMarshaler
func (p PathEdges) MarshalText() ([]byte, error) {
	return json.Marshal(PathEdgesText{p[0], p[1]})
}

// UnmarshalText implements json.Unmarshaler
func (p *PathEdges) UnmarshalText(b []byte) error {
	edges := PathEdgesText{}
	err := json.Unmarshal(b, &edges)
	if err != nil {
		return err
	}

	p[0] = edges.Edge1
	p[1] = edges.Edge2
	return nil
}

// Path is a list of hops between nodes (transports), and indicates a route between the edges
type Path []Hop

// Route is a succession of transport entries that denotes a path from source node to destination node
type Route []*Hop

func (r Route) String() string {
	res := "\n"
	for _, hop := range r {
		res += fmt.Sprintf("\t%s\n", hop)
	}

	return res
}
