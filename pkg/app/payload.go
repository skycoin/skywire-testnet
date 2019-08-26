package app

import (
	"encoding/json"
	"fmt"

	"github.com/skycoin/skywire/pkg/routing"
)

// Payload type encapsulates all possible payloads
type Payload struct {
	Frame Frame
	Data  []byte
}

func (p Payload) String() string {
	switch p.Frame {
	case FrameInit:
		var config Config
		if err := json.Unmarshal(p.Data, &config); err != nil {
			return fmt.Sprintf("{FrameInit. Error: %v, Data: %v}", err, p.Data)
		}
		return fmt.Sprintf("{FrameInit: %v}", config)
	case FrameConfirmLoop:
		var addrs [2]routing.Addr
		if err := json.Unmarshal(p.Data, &addrs); err != nil {
			return fmt.Sprintf("{FrameConfirmLoop. Error: %v, Data: %v}", err, p.Data)
		}
		return fmt.Sprintf("{FrameConfirmLoop. %v}", addrs)
	case FrameSend:
		packet := &Packet{}
		if err := json.Unmarshal(p.Data, packet); err != nil {
			return fmt.Sprintf("{FrameSend. Error: %v, data: %s}", err, p.Data)
		}
		return fmt.Sprintf("{FrameSend: %s}", packet)
	case FrameClose:
		var loop routing.AddressPair
		if err := json.Unmarshal(p.Data, &loop); err != nil {
			return fmt.Sprintf("{FrameClose. Error: %v, data: %v}", err, p.Data)
		}
		return fmt.Sprintf("{FrameClose: %v}", p.Data)
	case FrameFailure:
		return fmt.Sprintf("{FrameFailure: %s}", string(p.Data))
	case FrameCreateLoop:
		var raddr routing.Addr
		if err := json.Unmarshal(p.Data, &raddr); err != nil {
			return fmt.Sprintf("{FrameCreateLoop: Error: %v. Data: %v}", err, p.Data)
		}
		return fmt.Sprintf("{FrameCreateLoop: %s}", raddr)
	case FrameSuccess:
		return "{FrameSuccess}"
	default:
		return fmt.Sprintf("{Frame: %d. Data: %v}", p.Frame, p.Data)
	}

}
