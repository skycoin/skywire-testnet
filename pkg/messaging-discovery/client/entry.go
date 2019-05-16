package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"strings"

	"github.com/skycoin/skywire/pkg/cipher"
)

const currentVersion = "0.0.1"

// nolint
var (
	ErrKeyNotFound                = errors.New("entry of public key is not found")
	ErrUnexpected                 = errors.New("something unexpected happened")
	ErrUnauthorized               = errors.New("invalid signature")
	ErrBadInput                   = errors.New("error bad input")
	ErrValidationNonZeroSequence  = NewEntryValidationError("new entry has non-zero sequence")
	ErrValidationNilEphemerals    = NewEntryValidationError("entry of client instance has nil ephemeral keys")
	ErrValidationNilKeys          = NewEntryValidationError("entry Keys is nil")
	ErrValidationNonNilEphemerals = NewEntryValidationError("entry of server instance has non nil Keys.Ephemerals field")
	ErrValidationNoSignature      = NewEntryValidationError("entry has no signature")
	ErrValidationNoVersion        = NewEntryValidationError("entry has no version")
	ErrValidationNoClientOrServer = NewEntryValidationError("entry has neither client or server field")
	ErrValidationWrongSequence    = NewEntryValidationError("sequence field of new entry is not sequence of old entry + 1")
	ErrValidationWrongTime        = NewEntryValidationError("previous entry timestamp is not set before current entry timestamp")

	errReverseMap = map[string]error{
		ErrKeyNotFound.Error():                ErrKeyNotFound,
		ErrUnexpected.Error():                 ErrUnexpected,
		ErrUnauthorized.Error():               ErrUnauthorized,
		ErrBadInput.Error():                   ErrBadInput,
		ErrValidationNonZeroSequence.Error():  ErrValidationNonZeroSequence,
		ErrValidationNilEphemerals.Error():    ErrValidationNilEphemerals,
		ErrValidationNilKeys.Error():          ErrValidationNilKeys,
		ErrValidationNonNilEphemerals.Error(): ErrValidationNonNilEphemerals,
		ErrValidationNoSignature.Error():      ErrValidationNoSignature,
		ErrValidationNoVersion.Error():        ErrValidationNoVersion,
		ErrValidationNoClientOrServer.Error(): ErrValidationNoClientOrServer,
		ErrValidationWrongSequence.Error():    ErrValidationWrongSequence,
		ErrValidationWrongTime.Error():        ErrValidationWrongTime,
	}
)

func errFromString(s string) error {
	err, ok := errReverseMap[s]
	if !ok {
		return ErrUnexpected
	}
	return err
}

// EntryValidationError represents transient error caused by invalid
// data in Entry
type EntryValidationError struct {
	Cause string
}

// NewEntryValidationError constructs a new validation error.
func NewEntryValidationError(cause string) error {
	return EntryValidationError{cause}
}

func (e EntryValidationError) Error() string {
	return fmt.Sprintf("entry validation error: %s", e.Cause)
}

// Entry represents a Messaging Node's entry in the Discovery database.
type Entry struct {
	// The data structure's version.
	Version string `json:"version"`

	// An Entry of a given public key may need to iterate. This is the iteration sequence.
	Sequence uint64 `json:"sequence"`

	// Timestamp of the current iteration.
	Timestamp int64 `json:"timestamp"`

	// Static public key of an instance.
	Static cipher.PubKey `json:"static"`

	// Contains the instance's client meta if it's to be advertised as a Messaging Client.
	Client *Client `json:"client,omitempty"`

	// Contains the instance's server meta if it's to be advertised as a Messaging Server.
	Server *Server `json:"server,omitempty"`

	// Signature for proving authenticity of an Entry.
	Signature string `json:"signature,omitempty"`
}

func (e *Entry) String() string {
	res := ""
	res += fmt.Sprintf("\tversion: %s\n", e.Version)
	res += fmt.Sprintf("\tsequence: %d\n", e.Sequence)
	res += fmt.Sprintf("\tregistered at: %d\n", e.Timestamp)
	res += fmt.Sprintf("\tstatic public key: %s\n", e.Static)
	res += fmt.Sprintf("\tsignature: %s\n", e.Signature)

	if e.Client != nil {
		indentedStr := strings.Replace(e.Client.String(), "\n\t", "\n\t\t\t", -1)
		res += fmt.Sprintf("\tentry is registered as client. Related info: \n\t\t%s\n", indentedStr)
	}

	if e.Server != nil {
		indentedStr := strings.Replace(e.Server.String(), "\n\t", "\n\t\t", -1)
		res += fmt.Sprintf("\tentry is registered as server. Related info: \n\t%s\n", indentedStr)
	}

	return res
}

// Client contains parameters for Client instances.
type Client struct {
	// DelegatedServers contains a list of delegated servers represented by their public keys.
	DelegatedServers []cipher.PubKey `json:"delegated_servers"`
}

// String implements stringer
func (c *Client) String() string {
	res := "delegated servers: \n"

	for _, ds := range c.DelegatedServers {
		res += fmt.Sprintf("\t%s\n", ds)
	}

	return res
}

// Server contains parameters for Server instances.
type Server struct {
	// IPv4 or IPv6 public address of the Messaging Server.
	Address string `json:"address"`

	// Port in which the Messaging Server is listening for connections.
	Port string `json:"port"`

	// Number of connections still available.
	AvailableConnections int `json:"available_connections"`
}

// String implements stringer
func (s *Server) String() string {
	res := fmt.Sprintf("\taddress: %s\n", s.Address)
	res += fmt.Sprintf("\tport: %s\n", s.Port)
	res += fmt.Sprintf("\tavailable connections: %d\n", s.AvailableConnections)

	return res
}

// NewClientEntry is a convenience function that returns a valid client entry, but this entry
// should be signed with the private key before sending it to the server
func NewClientEntry(pubkey cipher.PubKey, sequence uint64, delegatedServers []cipher.PubKey) *Entry {
	return &Entry{
		Version:   currentVersion,
		Sequence:  sequence,
		Client:    &Client{delegatedServers},
		Static:    pubkey,
		Timestamp: time.Now().UnixNano(),
	}
}

// NewServerEntry constructs a new Server entry.
func NewServerEntry(pubkey cipher.PubKey, sequence uint64, address string, conns int) *Entry {
	return &Entry{
		Version:   currentVersion,
		Sequence:  sequence,
		Server:    &Server{Address: address, AvailableConnections: conns},
		Static:    pubkey,
		Timestamp: time.Now().UnixNano(),
	}
}

// VerifySignature check if signature matches to Entry's PubKey.
func (e *Entry) VerifySignature() error {
	entry := *e

	// Get and parse signature
	signature := cipher.Sig{}
	err := signature.UnmarshalText([]byte(e.Signature))
	if err != nil {
		return err
	}

	// Set signature field to zero-value
	entry.Signature = ""

	// Get hash of the entry
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return cipher.VerifyPubKeySignedPayload(cipher.PubKey(e.Static), signature, entryJSON)
}

// Sign signs Entry with provided SecKey.
func (e *Entry) Sign(sk cipher.SecKey) error {
	// Clear previous signature, in case there was any
	e.Signature = ""

	entryJSON, err := json.Marshal(e)
	if err != nil {
		return err
	}

	sig, err := cipher.SignPayload(entryJSON, cipher.SecKey(sk))
	if err != nil {
		return err
	}
	e.Signature = sig.Hex()
	return nil
}

// Validate checks if entry is valid.
func (e *Entry) Validate() error {
	// Must have version
	if e.Version == "" {
		return ErrValidationNoVersion
	}
	// Must be signed
	if e.Signature == "" {
		return ErrValidationNoSignature
	}

	// A record must have either client or server record
	if e.Client == nil && e.Server == nil {
		return ErrValidationNoClientOrServer
	}

	// The Keys field must exist
	if e.Static.Null() {
		return ErrValidationNilKeys
	}

	return nil
}

// ValidateIteration verifies Entry's Sequence against nextEntry.
func (e *Entry) ValidateIteration(nextEntry *Entry) error {

	// Sequence value must be {previous_sequence} + 1
	if e.Sequence != nextEntry.Sequence-1 {
		return ErrValidationWrongSequence
	}

	currentTimeStamp := time.Unix(e.Timestamp, 0)

	nextTimeStamp := time.Unix(nextEntry.Timestamp, 0)

	if !currentTimeStamp.Before(nextTimeStamp) {
		return ErrValidationWrongTime
	}

	return nil
}

// Copy performs a deep copy of two entries. It is safe to use with empty entries
func Copy(dst, src *Entry) {
	if dst.Server == nil && src.Server != nil {
		dst.Server = &Server{}
	}
	if dst.Client == nil && src.Client != nil {
		dst.Client = &Client{}
	}

	if src.Server == nil {
		dst.Server = nil
	} else {
		*dst.Server = *src.Server
	}
	if src.Client == nil {
		dst.Client = nil
	} else {
		*dst.Client = *src.Client
	}

	dst.Static = src.Static
	dst.Signature = src.Signature
	dst.Version = src.Version
	dst.Sequence = src.Sequence
	dst.Timestamp = src.Timestamp
}
