// Package setup defines setup node protocol.
package setup

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
)

// Packet defines type of a setup packet
type Packet byte

func (sp Packet) String() string {
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
	PacketAddRules Packet = iota
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

	// ResponseFailure represents failure response for a foundation packet.
	ResponseFailure = 0xfe
	// ResponseSuccess represents successful response for a foundation packet.
	ResponseSuccess = 0xff
)

// LoopData stores loop confirmation request data.
type LoopData struct {
	RemotePK     cipher.PubKey   `json:"remote-pk"`
	RemotePort   uint16          `json:"remote-port"`
	LocalPort    uint16          `json:"local-port"`
	RouteID      routing.RouteID `json:"resp-rid,omitempty"`
	NoiseMessage []byte          `json:"noise-msg,omitempty"`
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
func (p *Protocol) ReadPacket() (Packet, []byte, error) {
	frame, err := p.readFrame()
	if err != nil {
		return 0, nil, err
	}

	return Packet(frame[0]), frame[1:], nil
}

// Respond sends response to the remote node.
func (p *Protocol) Respond(res interface{}) error {
	if err, ok := res.(error); ok {
		return p.sendCMD(ResponseFailure, err)
	}

	return p.sendCMD(ResponseSuccess, res)
}

// AddRule sends AddRule setup request.
func (p *Protocol) AddRule(rule routing.Rule) (routeID routing.RouteID, err error) {
	if err = p.sendCMD(PacketAddRules, []routing.Rule{rule}); err != nil {
		return 0, err
	}

	res := []routing.RouteID{}
	if err = p.readRes(&res); err != nil {
		return 0, err
	}

	if len(res) == 0 {
		return 0, errors.New("empty response")
	}

	return res[0], nil
}

// DeleteRule sends DeleteRule setup request.
func (p *Protocol) DeleteRule(routeID routing.RouteID) error {
	if err := p.sendCMD(PacketDeleteRules, []routing.RouteID{routeID}); err != nil {
		return err
	}

	res := []routing.RouteID{}
	if err := p.readRes(&res); err != nil {
		return err
	}

	if len(res) == 0 {
		return errors.New("empty response")
	}

	return nil
}

// CreateLoop sends CreateLoop setup request.
func (p *Protocol) CreateLoop(l *routing.Loop) error {
	if err := p.sendCMD(PacketCreateLoop, l); err != nil {
		return err
	}

	if err := p.readRes(nil); err != nil {
		return err
	}

	return nil
}

// ConfirmLoop sends ConfirmLoop setup request.
func (p *Protocol) ConfirmLoop(l *LoopData) (noiseRes []byte, err error) {
	if err = p.sendCMD(PacketConfirmLoop, l); err != nil {
		return
	}

	res := []byte{}
	if err = p.readRes(&res); err != nil {
		return
	}

	return res, nil
}

// CloseLoop sends CloseLoop setup request.
func (p *Protocol) CloseLoop(l *LoopData) error {
	if err := p.sendCMD(PacketCloseLoop, l); err != nil {
		return err
	}

	if err := p.readRes(nil); err != nil {
		return err
	}

	return nil
}

// LoopClosed sends LoopClosed setup request.
func (p *Protocol) LoopClosed(l *LoopData) error {
	if err := p.sendCMD(PacketLoopClosed, l); err != nil {
		return err
	}

	if err := p.readRes(nil); err != nil {
		return err
	}

	return nil
}

func (p *Protocol) readRes(payload interface{}) error {
	frame, err := p.readFrame()
	if err != nil {
		return err
	}

	if frame[0] == ResponseFailure {
		return errors.New(string(frame[1:]))
	}

	if payload == nil {
		return nil
	}

	if err = json.Unmarshal(frame[1:], payload); err != nil {
		return err
	}

	return nil
}

func (p *Protocol) readFrame() ([]byte, error) {
	size := make([]byte, 2)
	if _, err := io.ReadFull(p.rw, size); err != nil {
		return nil, err
	}

	frame := make([]byte, binary.BigEndian.Uint16(size))
	_, err := io.ReadFull(p.rw, frame)
	if err != nil {
		return nil, err
	}

	if len(frame) == 0 {
		return nil, errors.New("empty frame")
	}

	return frame, nil
}

func (p *Protocol) sendCMD(cmdType Packet, payload interface{}) error {
	var data []byte
	if err, ok := payload.(error); ok {
		data = []byte(err.Error())
	} else {
		data, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	packet := append([]byte{byte(cmdType)}, data...)
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(packet)))
	_, err := p.rw.Write(append(buf, packet...))
	return err
}
