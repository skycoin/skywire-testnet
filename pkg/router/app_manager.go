package router

import (
	"encoding/json"
	"errors"

	"github.com/skycoin/skywire/internal/appnet"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
)

const supportedProtocolVersion = "0.0.1"

type appCallbacks struct {
	CreateLoop func(conn *appnet.Protocol, raddr *appnet.LoopAddr) (laddr *appnet.LoopAddr, err error)
	CloseLoop  func(conn *appnet.Protocol, addr *app.LoopMeta) error
	Forward    func(conn *appnet.Protocol, packet *app.DataFrame) error
}

type appManager struct {
	Logger *logging.Logger

	proto     *appnet.Protocol
	appConf   *app.Config
	callbacks *appCallbacks
}

func (am *appManager) Serve() error {
	return am.proto.ServeJSON(func(frame appnet.FrameType, payload []byte) (res interface{}, err error) {
		am.Logger.Infof("Got new App request with type %s: %s", frame, string(payload))
		switch frame {
		case appnet.FrameCreateLoop:
			res, err = am.setupLoop(payload)
		case appnet.FrameCloseLoop:
			err = am.handleCloseLoop(payload)
		case appnet.FrameData:
			err = am.forwardAppPacket(payload)
		default:
			err = errors.New("unexpected frame")
		}

		if err != nil {
			am.Logger.Infof("App request with type %s failed: %s", frame, err)
		}

		return res, err
	})
}

func (am *appManager) setupLoop(payload []byte) (*appnet.LoopAddr, error) {
	raddr := &app.LoopAddr{}
	if err := json.Unmarshal(payload, raddr); err != nil {
		return nil, err
	}

	return am.callbacks.CreateLoop(am.proto, raddr)
}

func (am *appManager) handleCloseLoop(payload []byte) error {
	addr := &app.LoopMeta{}
	if err := json.Unmarshal(payload, addr); err != nil {
		return err
	}

	return am.callbacks.CloseLoop(am.proto, addr)
}

func (am *appManager) forwardAppPacket(payload []byte) error {
	packet := &app.DataFrame{}
	if err := json.Unmarshal(payload, packet); err != nil {
		return err
	}

	return am.callbacks.Forward(am.proto, packet)
}
