package transport

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
)

// Entry is the unsigned representation of a Transport.
type Entry struct {

	// ID is the Transport ID that uniquely identifies the Transport.
	ID uuid.UUID `json:"t_id"`

	// Edges contains the public keys of the Transport's edge nodes (should only have 2 edges and the least-significant edge should come first).
	Edges [2]cipher.PubKey `json:"edges"`

	// Type represents the transport type.
	Type string `json:"type"`

	// Public determines whether the transport is to be exposed to other nodes or not.
	// Public transports are to be registered in the Transport Discovery.
	Public bool `json:"public"` // TODO: remove this.
}

// NewEntry constructs *Entry
func NewEntry(localPK, remotePK cipher.PubKey, tpType string, public bool) *Entry {
	return &Entry{
		ID:     MakeTransportID(localPK, remotePK, tpType),
		Edges:  SortEdges(localPK, remotePK),
		Type:   tpType,
		Public: public,
	}
}

// SetEdges sets edges of Entry
func (e *Entry) SetEdges(localPK, remotePK cipher.PubKey) {
	e.ID = MakeTransportID(localPK, remotePK, e.Type)
	e.Edges = SortEdges(localPK, remotePK)
}

// RemoteEdge returns the remote edge's public key.
func (e *Entry) RemoteEdge(local cipher.PubKey) cipher.PubKey {
	for _, pk := range e.Edges {
		if pk != local {
			return pk
		}
	}
	return local
}

// EdgeIndex returns the index location of the given public key.
// Returns -1 if the edge is not found.
func (e *Entry) EdgeIndex(pk cipher.PubKey) int {
	for i, edgePK := range e.Edges {
		if pk == edgePK {
			return i
		}
	}
	return -1
}

// HasEdge returns true if the provided edge is present in 'e.Edges' field.
func (e *Entry) HasEdge(edge cipher.PubKey) bool {
	for _, pk := range e.Edges {
		if pk == edge {
			return true
		}
	}
	return false
}

// String implements stringer
func (e *Entry) String() string {
	res := ""
	if e.Public {
		res += fmt.Sprintf("visibility: public\n")
	} else {
		res += fmt.Sprintf("visibility: private\n")
	}
	res += fmt.Sprintf("\ttype: %s\n", e.Type)
	res += fmt.Sprintf("\tid: %s\n", e.ID)
	res += fmt.Sprintf("\tedges:\n")
	res += fmt.Sprintf("\t\tedge 1: %s\n", e.Edges[0])
	res += fmt.Sprintf("\t\tedge 2: %s\n", e.Edges[1])
	return res
}

// ToBinary returns binary representation of an Entry
func (e *Entry) ToBinary() []byte {
	bEntry := e.ID[:]
	for _, edge := range e.Edges {
		bEntry = append(bEntry, edge[:]...)
	}
	return append(bEntry, []byte(e.Type)...)
}

// Signature returns signature for Entry calculated from binary
// representation.
func (e *Entry) Signature(secKey cipher.SecKey) cipher.Sig {
	sig, err := cipher.SignPayload(e.ToBinary(), secKey)
	if err != nil {
		panic(err)
	}
	return sig
}

// SignedEntry holds an Entry and it's associated signatures.
// The signatures should be ordered as the contained 'Entry.Edges'.
type SignedEntry struct {
	Entry      *Entry        `json:"entry"`
	Signatures [2]cipher.Sig `json:"signatures"`
	Registered int64         `json:"registered,omitempty"`
}

// Sign sets Signature for a given PubKey in correct position
func (se *SignedEntry) Sign(pk cipher.PubKey, secKey cipher.SecKey) bool {
	idx := se.Entry.EdgeIndex(pk)
	if idx == -1 {
		return false
	}
	se.Signatures[idx] = se.Entry.Signature(secKey)

	return true
}

// Signature gets Signature for a given PubKey from correct position
func (se *SignedEntry) Signature(pk cipher.PubKey) (cipher.Sig, bool) {
	idx := se.Entry.EdgeIndex(pk)
	if idx == -1 {
		return cipher.Sig{}, false
	}
	return se.Signatures[idx], true
}

// NewSignedEntry creates a SignedEntry with first signature
func NewSignedEntry(entry *Entry, pk cipher.PubKey, secKey cipher.SecKey) (*SignedEntry, bool) {
	se := &SignedEntry{Entry: entry}
	return se, se.Sign(pk, secKey)

}

// Status represents the current state of a Transport from a Transport's single edge.
// Each Transport will have two perspectives; one from each of it's edges.
type Status struct {

	// ID is the Transport ID that identifies the Transport that this status is regarding.
	ID uuid.UUID `json:"t_id"`

	// IsUp represents whether the Transport is up.
	// A Transport that is down will fail to forward Packets.
	IsUp bool `json:"is_up"`

	// Updated is the epoch timestamp of when the status is last updated.
	Updated int64 `json:"updated,omitempty"`
}

// EntryWithStatus stores Entry and Statuses returned by both Edges.
type EntryWithStatus struct {
	Entry      *Entry  `json:"entry"`
	IsUp       bool    `json:"is_up"`
	Registered int64   `json:"registered"`
	Statuses   [2]bool `json:"statuses"`
}

// String implements stringer
func (e *EntryWithStatus) String() string {
	res := "entry:\n"
	res += fmt.Sprintf("\tregistered at: %d\n", e.Registered)
	res += fmt.Sprintf("\tstatus returned by edge 1: %t\n", e.Statuses[0])
	res += fmt.Sprintf("\tstatus returned by edge 2: %t\n", e.Statuses[1])
	if e.IsUp {
		res += fmt.Sprintf("\ttransport: up\n")
	} else {
		res += fmt.Sprintf("\ttransport: down\n")
	}
	indentedStr := strings.Replace(e.Entry.String(), "\n\t", "\n\t\t", -1)
	res += fmt.Sprintf("\ttransport info: \n\t\t%s", indentedStr)

	return res
}
