// Package setup defines setup node protocol.
package setup

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

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
		return "OnConfirmLoop"
	case PacketCloseLoop:
		return "CloseLoop"
	case PacketLoopClosed:
		return "OnLoopClosed"
	case RespSuccess:
		return "Success"
	case RespFailure:
		return "Failure"
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
	// PacketConfirmLoop represents OnConfirmLoop foundation packet.
	PacketConfirmLoop
	// PacketCloseLoop represents CloseLoop foundation packet.
	PacketCloseLoop
	// PacketLoopClosed represents OnLoopClosed foundation packet.
	PacketLoopClosed

	// RespFailure represents failure response for a foundation packet.
	RespFailure = 0xfe
	// RespSuccess represents successful response for a foundation packet.
	RespSuccess = 0xff
)

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
	h := make([]byte, 3)
	if _, err := io.ReadFull(p.rw, h); err != nil {
		return 0, nil, err
	}
	t := PacketType(h[0])
	pay := make([]byte, binary.BigEndian.Uint16(h[1:3]))
	if _, err := io.ReadFull(p.rw, pay); err != nil {
		return 0, nil, err
	}
	if len(pay) == 0 {
		return 0, nil, errors.New("empty packet")
	}
	//fmt.Println(p.pks(), "READ:", t, string(pay))
	return t, pay, nil
}

// WritePacket writes a single setup packet.
func (p *Protocol) WritePacket(t PacketType, body interface{}) error {
	pay, err := json.Marshal(body)
	if err != nil {
		return err
	}
	//fmt.Println(p.pks(), "WRITE:", t, string(pay))
	raw := make([]byte, 3+len(pay))
	raw[0] = byte(t)
	binary.BigEndian.PutUint16(raw[1:3], uint16(len(pay)))
	copy(raw[3:], pay)
	_, err = p.rw.Write(raw)
	return err
}

// AddRule sends AddRule setup request.
func AddRule(ctx context.Context, p *Protocol, rule routing.Rule) (routeID routing.RouteID, err error) {
	if err = p.WritePacket(PacketAddRules, []routing.Rule{rule}); err != nil {
		return 0, err
	}
	var res []routing.RouteID
	if err = readAndDecodePacketWithTimeout(ctx, p, &res); err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, errors.New("empty response")
	}
	return res[0], nil
}

// DeleteRule sends DeleteRule setup request.
func DeleteRule(ctx context.Context, p *Protocol, routeID routing.RouteID) error {
	if err := p.WritePacket(PacketDeleteRules, []routing.RouteID{routeID}); err != nil {
		return err
	}
	var res []routing.RouteID
	if err := readAndDecodePacketWithTimeout(ctx, p, &res); err != nil {
		return err
	}
	if len(res) == 0 {
		return errors.New("empty response")
	}
	return nil
}

// CreateLoop sends CreateLoop setup request.
func CreateLoop(ctx context.Context, p *Protocol, ld routing.LoopDescriptor) error {
	if err := p.WritePacket(PacketCreateLoop, ld); err != nil {
		return err
	}
	return readAndDecodePacketWithTimeout(ctx, p, nil) // TODO: data race.
}

// OnConfirmLoop sends OnConfirmLoop setup request.
func ConfirmLoop(ctx context.Context, p *Protocol, ld routing.LoopData) error {
	if err := p.WritePacket(PacketConfirmLoop, ld); err != nil {
		return err
	}
	return readAndDecodePacketWithTimeout(ctx, p, nil)
}

// CloseLoop sends CloseLoop setup request.
func CloseLoop(ctx context.Context, p *Protocol, ld routing.LoopData) error {
	if err := p.WritePacket(PacketCloseLoop, ld); err != nil {
		return err
	}
	return readAndDecodePacketWithTimeout(ctx, p, nil)
}

// OnLoopClosed sends OnLoopClosed setup request.
func LoopClosed(ctx context.Context, p *Protocol, ld routing.LoopData) error {
	if err := p.WritePacket(PacketLoopClosed, ld); err != nil {
		return err
	}
	return readAndDecodePacketWithTimeout(ctx, p, nil)
}

func readAndDecodePacketWithTimeout(ctx context.Context, p *Protocol, v interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, ReadTimeout)
	defer cancel()

	done := make(chan struct{})
	var err error
	go func() {
		err = readAndDecodePacket(p, v)
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return err
	}
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
	return json.Unmarshal(raw, v)
}
