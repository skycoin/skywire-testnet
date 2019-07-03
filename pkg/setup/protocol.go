// Package setup defines setup node protocol.
package setup

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/routing"
)

// PacketType defines type of a setup packet
type PacketType byte

func (sp PacketType) String() string {
	switch sp {
	case PacketAddRules:
		return "AddRules"
	case PacketDeleteRules:
		return "DeleteRules"
	case PacketCreateLoop:
		return "CreateLoop"
	case PacketConfirmLoop:
		return "ConfirmLoop"
	case PacketCloseLoop:
		return "CloseLoop"
	case PacketLoopClosed:
		return "LoopClosed"
	}
	return fmt.Sprintf("Unknown(%d)", sp)
}

const (
	// PacketAddRules represents AddRules foundation packet.
	PacketAddRules PacketType = iota
	// PacketDeleteRules represents DeleteRules foundation packet.
	PacketDeleteRules
	// PacketCreateLoop represents CreateLoop foundation packet.
	PacketCreateLoop
	// PacketConfirmLoop represents ConfirmLoop foundation packet.
	PacketConfirmLoop
	// PacketCloseLoop represents CloseLoop foundation packet.
	PacketCloseLoop
	// PacketLoopClosed represents LoopClosed foundation packet.
	PacketLoopClosed

	// RespFailure represents failure response for a foundation packet.
	RespFailure = 0xfe
	// RespSuccess represents successful response for a foundation packet.
	RespSuccess = 0xff
)

// LoopData stores loop confirmation request data.
type LoopData struct {
	RemotePK   cipher.PubKey   `json:"remote-pk"`
	RemotePort uint16          `json:"remote-port"`
	LocalPort  uint16          `json:"local-port"`
	RouteID    routing.RouteID `json:"resp-rid,omitempty"`
}

// Protocol defines routes setup protocol.
type Protocol struct {
	rw io.ReadWriter
}

// NewSetupProtocol constructs a new setup Protocol.
func NewSetupProtocol(rw io.ReadWriter) *Protocol {
	return &Protocol{rw}
}

// ReadPacket reads a single setup packet.
func (p *Protocol) ReadPacket() (PacketType, []byte, error) {
	rawLen := make([]byte, 2)
	if _, err := io.ReadFull(p.rw, rawLen); err != nil { // TODO: data race.
		return 0, nil, err
	}
	rawBody := make([]byte, binary.BigEndian.Uint16(rawLen))
	_, err := io.ReadFull(p.rw, rawBody)
	if err != nil {
		return 0, nil, err
	}
	if len(rawBody) == 0 {
		return 0, nil, errors.New("empty packet")
	}
	return PacketType(rawBody[0]), rawBody[1:], nil
}

// WritePacket writes a single setup packet.
func (p *Protocol) WritePacket(t PacketType, body interface{}) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.Grow(3 + len(raw))
	if err := binary.Write(&buf, binary.BigEndian, uint16(1+len(raw))); err != nil {
		return err
	}
	if err := buf.WriteByte(byte(t)); err != nil {
		return err
	}
	if _, err := buf.Write(raw); err != nil {
		return err
	}
	_, err = buf.WriteTo(p.rw)
	return err
}

// AddRule sends AddRule setup request.
func AddRule(p *Protocol, rule routing.Rule) (routeID routing.RouteID, err error) {
	if err = p.WritePacket(PacketAddRules, []routing.Rule{rule}); err != nil {
		return 0, err
	}
	var res []routing.RouteID
	if err = readAndDecodePacket(p, &res); err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, errors.New("empty response")
	}
	return res[0], nil
}

// DeleteRule sends DeleteRule setup request.
func DeleteRule(p *Protocol, routeID routing.RouteID) error {
	if err := p.WritePacket(PacketDeleteRules, []routing.RouteID{routeID}); err != nil {
		return err
	}
	var res []routing.RouteID
	if err := readAndDecodePacket(p, &res); err != nil {
		return err
	}
	if len(res) == 0 {
		return errors.New("empty response")
	}
	return nil
}

// CreateLoop sends CreateLoop setup request.
func CreateLoop(p *Protocol, l *routing.Loop) error {
	if err := p.WritePacket(PacketCreateLoop, l); err != nil {
		return err
	}
	if err := readAndDecodePacket(p, nil); err != nil { // TODO: data race.
		return err
	}
	return nil
}

// ConfirmLoop sends ConfirmLoop setup request.
func ConfirmLoop(p *Protocol, l *LoopData) error {
	if err := p.WritePacket(PacketConfirmLoop, l); err != nil {
		return err
	}
	var res []byte
	return readAndDecodePacket(p, &res)
}

// CloseLoop sends CloseLoop setup request.
func CloseLoop(p *Protocol, l *LoopData) error {
	if err := p.WritePacket(PacketCloseLoop, l); err != nil {
		return err
	}
	if err := readAndDecodePacket(p, nil); err != nil {
		return err
	}
	return nil
}

// LoopClosed sends LoopClosed setup request.
func LoopClosed(p *Protocol, l *LoopData) error {
	if err := p.WritePacket(PacketLoopClosed, l); err != nil {
		return err
	}
	if err := readAndDecodePacket(p, nil); err != nil {
		return err
	}
	return nil
}

func readAndDecodePacket(p *Protocol, v interface{}) error {
	t, raw, err := p.ReadPacket() // TODO: data race.
	if err != nil {
		return err
	}
	if t == RespFailure {
		return errors.New("RespFailure, packet type: " + t.String())
	}
	if v == nil {
		return nil
	}
	if err = json.Unmarshal(raw, v); err != nil {
		return err
	}
	return nil
}
