package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/SkycoinProject/skycoin/src/util/logging"

	"github.com/SkycoinProject/skywire-mainnet/pkg/app"
	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
)

const supportedProtocolVersion = "0.0.1"

type appCallbacks struct {
	CreateLoop func(ctx context.Context, conn *app.Protocol, raddr routing.Addr) (laddr routing.Addr, err error)
	CloseLoop  func(ctx context.Context, conn *app.Protocol, loop routing.Loop) error
	Forward    func(ctx context.Context, conn *app.Protocol, packet *app.Packet) error
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

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		switch frame {
		case app.FrameInit:
			err = am.initApp(payload)
		case app.FrameCreateLoop:
			res, err = am.setupLoop(ctx, payload)
		case app.FrameClose:
			err = am.handleCloseLoop(ctx, payload)
		case app.FrameSend:
			err = am.forwardAppPacket(ctx, payload)
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
	var config app.Config
	if err := json.Unmarshal(payload, &config); err != nil {
		fmt.Println("invalid init:", string(payload))
		return fmt.Errorf("invalid INIT payload: %v", err)
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

	am.Logger.Infof("Finished initiating app: %s.v%s", config.AppName, config.AppVersion)
	return nil
}

func (am *appManager) setupLoop(ctx context.Context, payload []byte) (routing.Addr, error) {
	var raddr routing.Addr
	if err := json.Unmarshal(payload, &raddr); err != nil {
		return routing.Addr{}, err
	}
	return am.callbacks.CreateLoop(ctx, am.proto, raddr)
}

func (am *appManager) handleCloseLoop(ctx context.Context, payload []byte) error {
	var loop routing.Loop
	if err := json.Unmarshal(payload, &loop); err != nil {
		return err
	}
	return am.callbacks.CloseLoop(ctx, am.proto, loop)
}

func (am *appManager) forwardAppPacket(ctx context.Context, payload []byte) error {
	packet := &app.Packet{}
	if err := json.Unmarshal(payload, packet); err != nil {
		return err
	}
	return am.callbacks.Forward(ctx, am.proto, packet)
}
