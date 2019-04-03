package transport

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/cipher"
)

// Entry is the unsigned representation of a Transport.
type Entry struct {

	// ID is the Transport ID that uniquely identifies the Transport.
	ID uuid.UUID `json:"t_id"`

	// Edges contains the public keys of the Transport's edge nodes (should only have 2 edges and the least-significant edge should come first).
	EdgesKeys [2]cipher.PubKey `json:"edges"`

	// Type represents the transport type.
	Type string `json:"type"`

	// Public determines whether the transport is to be exposed to other nodes or not.
	// Public transports are to be registered in the Transport Discovery.
	Public bool `json:"public"`
}

// NewEntry constructs *Entry
func NewEntry(edgeA, edgeB cipher.PubKey, tpType string, public bool) *Entry {
	return &Entry{
		ID:        GetTransportUUID(edgeA, edgeB, tpType),
		EdgesKeys: SortPubKeys(edgeA, edgeB),
		Type:      tpType,
		Public:    public,
	}
}

// Edges returns edges of Entry
func (e *Entry) Edges() [2]cipher.PubKey {
	// this sort *must* be needless
	// but to remove it:
	// - all tests must be passed
	// - written Benchmarks
	return SortPubKeys(e.EdgesKeys[0], e.EdgesKeys[1])
}

// SetEdges sets edges of Entry
func (e *Entry) SetEdges(edges [2]cipher.PubKey) {
	e.ID = GetTransportUUID(edges[0], edges[1], e.Type)
	e.EdgesKeys = SortPubKeys(edges[0], edges[1])
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
	res += fmt.Sprintf("\t\tedge 1: %s\n", e.Edges()[0])
	res += fmt.Sprintf("\t\tedge 2: %s\n", e.Edges()[1])

	return res
}

// ToBinary returns binary representation of a Signature.
func (e *Entry) ToBinary() []byte {
	bEntry := e.ID[:]
	for _, edge := range e.Edges() {
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

func (se *SignedEntry) Index(pk cipher.PubKey) byte {
	if pk == se.Entry.Edges()[1] {
		return 1
	}
	return 0
}

// SetSignature sets Signature for a given PubKey in correct position
func (se *SignedEntry) SetSignature(pk cipher.PubKey, secKey cipher.SecKey) {
	idx := se.Index(pk)
	se.Signatures[idx] = se.Entry.Signature(secKey)
}

// GetSignature gets Signature for a given PubKey from correct position
func (se *SignedEntry) GetSignature(pk cipher.PubKey) cipher.Sig {
	idx := se.Index(pk)
	return se.Signatures[idx]
}

// NewSignedEntry creates a SignedEntry with first signature
func NewSignedEntry(entry *Entry, pk cipher.PubKey, secKey cipher.SecKey) *SignedEntry {
	se := &SignedEntry{Entry: entry}
	se.SetSignature(pk, secKey)
	return se
}

// Status represents the current state of a Transport from the perspective
// from a Transport's single edge. Each Transport will have two perspectives;
// one from each of it's edges.
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
