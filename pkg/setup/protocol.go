// Package setup defines setup node protocol.
package setup

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/transport/dmsg"

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

// Protocol defines routes setup protocol.
type Protocol struct {
	rw io.ReadWriter
}

func (p *Protocol) pks() string {
	humanize := func(pk cipher.PubKey) string {
		switch pk.String() {
		case "02d75d089b307032a3dfd1e6808f6a4ea0011205f60e7f28f69f444c4e48b2b7c3":
			return "1"
		case "0217fcf0478d41aa7eab8ae023658910e76e5aeae9bf18fdd607188c290a9be649":
			return "2"
		case "028e90b6fcb4ce3ecdf0fff555be992ab612d85c00897af8bdabff157916468fc1":
			return "3"
		case "03739ff49de06eab6b26f55e658c97121457bb690a1d8b55cd892decbc95073b80":
			return "4"
		case "0345327088f0f34c359ecf67892fbe58a1cd7299e15a797339fa29365cc2e7c551":
			return "s"
		default:
			return fmt.Sprintf("(%s)", pk)
		}
	}
	tp, ok := p.rw.(*dmsg.Transport)
	if !ok {
		return "[~~]"
	}
	return fmt.Sprintf("[%s%s]", humanize(tp.LocalPK()), humanize(tp.RemotePK()))
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
func CreateLoop(p *Protocol, ld routing.LoopDescriptor) error {
	if err := p.WritePacket(PacketCreateLoop, ld); err != nil {
		return err
	}
	return readAndDecodePacket(p, nil) // TODO: data race.
}

// ConfirmLoop sends ConfirmLoop setup request.
func ConfirmLoop(p *Protocol, ld routing.LoopData) error {
	if err := p.WritePacket(PacketConfirmLoop, ld); err != nil {
		return err
	}
	return readAndDecodePacket(p, nil)
}

// CloseLoop sends CloseLoop setup request.
func CloseLoop(p *Protocol, ld routing.LoopData) error {
	if err := p.WritePacket(PacketCloseLoop, ld); err != nil {
		return err
	}
	return readAndDecodePacket(p, nil)
}

// LoopClosed sends LoopClosed setup request.
func LoopClosed(p *Protocol, ld routing.LoopData) error {
	if err := p.WritePacket(PacketLoopClosed, ld); err != nil {
		return err
	}
	return readAndDecodePacket(p, nil)
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
