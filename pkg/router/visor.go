package router

import (
	"encoding/json"
	"errors"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
)

const supportedProtocolVersion = "0.0.1"

type appCallbacks struct {
	CreateLoop func(conn *app.Protocol, raddr *app.Addr) (laddr *app.Addr, err error)
	CloseLoop  func(conn *app.Protocol, addr *app.LoopAddr) error
	Forward    func(conn *app.Protocol, packet *app.Packet) error
}

type visor struct {
	Logger *logging.Logger

	proto     *app.Protocol
	appConf   *app.Config
	callbacks *appCallbacks
}

func (v *visor) Serve() error {
	return v.proto.Serve(func(frame app.Frame, payload []byte) (res interface{}, err error) {
		v.Logger.Infof("Got new App request with type %s: %s", frame, string(payload))
		switch frame {
		case app.FrameInit:
			err = v.initApp(payload)
		case app.FrameCreateLoop:
			res, err = v.setupLoop(payload)
		case app.FrameClose:
			err = v.handleCloseLoop(payload)
		case app.FrameSend:
			err = v.forwardAppPacket(payload)
		default:
			err = errors.New("unexpected frame")
		}

		if err != nil {
			v.Logger.Infof("App request with type %s failed: %s", frame, err)
		}

		return res, err
	})
}

func (v *visor) initApp(payload []byte) error {
	config := &app.Config{}
	if err := json.Unmarshal(payload, config); err != nil {
		return errors.New("invalid Init payload")
	}

	if config.ProtocolVersion != supportedProtocolVersion {
		return errors.New("unsupported protocol version")
	}

	if v.appConf.AppName != config.AppName {
		return errors.New("unexpected app")
	}

	if v.appConf.AppVersion != config.AppVersion {
		return errors.New("unexpected app version")
	}

	v.Logger.Infof("Handshaked new connection with the app %s.v%s", config.AppName, config.AppVersion)
	return nil
}

func (v *visor) setupLoop(payload []byte) (*app.Addr, error) {
	raddr := &app.Addr{}
	if err := json.Unmarshal(payload, raddr); err != nil {
		return nil, err
	}

	return v.callbacks.CreateLoop(v.proto, raddr)
}

func (v *visor) handleCloseLoop(payload []byte) error {
	addr := &app.LoopAddr{}
	if err := json.Unmarshal(payload, addr); err != nil {
		return err
	}

	return v.callbacks.CloseLoop(v.proto, addr)
}

func (v *visor) forwardAppPacket(payload []byte) error {
	packet := &app.Packet{}
	if err := json.Unmarshal(payload, packet); err != nil {
		return err
	}

	return v.callbacks.Forward(v.proto, packet)
}
