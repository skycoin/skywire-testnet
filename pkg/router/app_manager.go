package router

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
)

const supportedProtocolVersion = "0.0.1"

type appCallbacks struct {
	CreateLoop func(conn *app.Protocol, raddr *app.Addr) (laddr *app.Addr, err error)
	CloseLoop  func(conn *app.Protocol, addr *app.LoopAddr) error
	Forward    func(conn *app.Protocol, packet *app.Packet) error
}

type appManager struct {
	Logger *logging.Logger

	proto     *app.Protocol
	appConf   *app.Config
	callbacks *appCallbacks
}

func (am *appManager) Serve() error {
	return am.proto.Serve(func(frame app.Frame, payload []byte) (res interface{}, err error) {
		am.Logger.Infof("Got new App request with type %s: %s", frame, string(payload))
		switch frame {
		case app.FrameInit:
			err = am.initApp(payload)
		case app.FrameCreateLoop:
			res, err = am.setupLoop(payload)
		case app.FrameClose:
			err = am.handleCloseLoop(payload)
		case app.FrameSend:
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

func (am *appManager) initApp(payload []byte) error {
	config := &app.Config{}
	if err := json.Unmarshal(payload, config); err != nil {
		return errors.New("invalid Init payload")
	}

	if config.ProtocolVersion != supportedProtocolVersion {
		return errors.New("unsupported protocol version")
	}

	if am.appConf.AppName != config.AppName {
		return errors.New("unexpected app")
	}

	if am.appConf.AppVersion != config.AppVersion {
		return errors.New("unexpected app version")
	}

	am.Logger.Infof("Handshaked new connection with the app %s.v%s", config.AppName, config.AppVersion)
	return nil
}

func (am *appManager) setupLoop(payload []byte) (*app.Addr, error) {
	raddr := &app.Addr{}
	if err := json.Unmarshal(payload, raddr); err != nil {
		return nil, err
	}

	return am.callbacks.CreateLoop(am.proto, raddr)
}

func (am *appManager) handleCloseLoop(payload []byte) error {
	addr := &app.LoopAddr{}
	if err := json.Unmarshal(payload, addr); err != nil {
		return err
	}

	return am.callbacks.CloseLoop(am.proto, addr)
}

func (am *appManager) forwardAppPacket(payload []byte) error {
	fmt.Println("am.forwardAppPacket() called")
	packet := &app.Packet{}
	if err := json.Unmarshal(payload, packet); err != nil {
		return err
	}

	return am.callbacks.Forward(am.proto, packet)
}
