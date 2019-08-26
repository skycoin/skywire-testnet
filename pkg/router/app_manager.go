package router

import (
	"encoding/json"
	"errors"

	"github.com/skycoin/skycoin/src/util/logging"

	th "github.com/skycoin/skywire/internal/testhelpers"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
)

const supportedProtocolVersion = "0.0.1"

type appManager struct {
	Logger  *logging.Logger
	router  PacketRouter
	proto   *app.Protocol
	appConf *app.Config
}

func (am *appManager) serveHandler(frame app.Frame, payload []byte) (res interface{}, err error) {
	am.Logger.WithField("payload", app.Payload{Frame: frame, Data: payload}).
		Debug(th.Trace("ENTER"))
	defer am.Logger.Info(th.Trace("EXIT"))

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
		am.Logger.Warnf("%v: App request with type %s failed: %+v\n", th.GetCaller(), frame, err)
	}

	return res, err
}

func (am *appManager) Serve() error {
	am.Logger.Info(th.Trace("ENTER"))
	defer am.Logger.Info(th.Trace("EXIT"))

	return am.proto.Serve(am.serveHandler)
}

func (am *appManager) initApp(payload []byte) error {
	am.Logger.Info(th.Trace("ENTER"))
	defer am.Logger.Info(th.Trace("EXIT"))

	var config app.Config
	if err := json.Unmarshal(payload, &config); err != nil {
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

	am.Logger.Infof("Finished initiating app: %s.v%s", config.AppName, config.AppVersion)
	return nil
}

func (am *appManager) setupLoop(payload []byte) (routing.Addr, error) {
	am.Logger.Info(th.Trace("ENTER"))
	defer am.Logger.Info(th.Trace("EXIT"))

	var raddr routing.Addr
	if err := json.Unmarshal(payload, &raddr); err != nil {
		return routing.Addr{}, err
	}

	return am.router.CreateLoop(am.proto, raddr)
}

func (am *appManager) handleCloseLoop(payload []byte) error {
	am.Logger.Info(th.Trace("ENTER"))
	defer am.Logger.Info(th.Trace("EXIT"))

	var loop routing.AddrLoop
	if err := json.Unmarshal(payload, &loop); err != nil {
		return err
	}

	return am.router.CloseLoop(am.proto, loop)
}

func (am *appManager) forwardAppPacket(payload []byte) error {
	am.Logger.Info(th.Trace("ENTER"))
	defer am.Logger.Info(th.Trace("EXIT"))

	packet := &app.Packet{}
	if err := json.Unmarshal(payload, packet); err != nil {
		return err
	}
	am.Logger.Debugf("%v: %v\n", th.GetCaller(), packet)

	return am.router.ForwardAppPacket(am.proto, packet)
}
