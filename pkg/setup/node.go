package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/SkycoinProject/dmsg"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/dmsg/disc"
	"github.com/SkycoinProject/skycoin/src/util/logging"

	"github.com/SkycoinProject/skywire-mainnet/pkg/metrics"
	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
	"github.com/SkycoinProject/skywire-mainnet/pkg/snet"
)

// Node performs routes setup operations over messaging channel.
type Node struct {
	Logger   *logging.Logger
	dmsgC    *dmsg.Client
	dmsgL    *dmsg.Listener
	srvCount int
	metrics  metrics.Recorder
}

// NewNode constructs a new SetupNode.
func NewNode(conf *Config, metrics metrics.Recorder) (*Node, error) {
	ctx := context.Background()

	logger := logging.NewMasterLogger()
	if lvl, err := logging.LevelFromString(conf.LogLevel); err == nil {
		logger.SetLevel(lvl)
	}
	log := logger.PackageLogger("setup_node")

	// Prepare dmsg.
	dmsgC := dmsg.NewClient(
		conf.PubKey,
		conf.SecKey,
		disc.NewHTTP(conf.Messaging.Discovery),
		dmsg.SetLogger(logger.PackageLogger(dmsg.Type)),
	)
	if err := dmsgC.InitiateServerConnections(ctx, conf.Messaging.ServerCount); err != nil {
		return nil, fmt.Errorf("failed to init dmsg: %s", err)
	}
	log.Info("connected to dmsg servers")

	dmsgL, err := dmsgC.Listen(snet.SetupPort)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on dmsg port %d: %v", snet.SetupPort, dmsgL)
	}
	log.Info("started listening for dmsg connections")

	return &Node{
		Logger:   log,
		dmsgC:    dmsgC,
		dmsgL:    dmsgL,
		srvCount: conf.Messaging.ServerCount,
		metrics:  metrics,
	}, nil
}

// Close closes underlying dmsg client.
func (sn *Node) Close() error {
	if sn == nil {
		return nil
	}
	return sn.dmsgC.Close()
}

// Serve starts transport listening loop.
func (sn *Node) Serve(ctx context.Context) error {
	sn.Logger.Info("serving setup node")

	for {
		conn, err := sn.dmsgL.AcceptTransport()
		if err != nil {
			return err
		}
		go func(conn *dmsg.Transport) {
			if err := sn.handleRequest(ctx, conn); err != nil {
				sn.Logger.Warnf("Failed to serve Transport: %s", err)
			}
		}(conn)
	}
}

func (sn *Node) handleRequest(ctx context.Context, tr *dmsg.Transport) error {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	proto := NewSetupProtocol(tr)
	sp, data, err := proto.ReadPacket()
	if err != nil {
		return err
	}

	log := sn.Logger.WithField("requester", tr.RemotePK()).WithField("reqType", sp)
	log.Infof("Received request.")

	startTime := time.Now()

	switch sp {
	case PacketCreateLoop:
		var ld routing.LoopDescriptor
		if err = json.Unmarshal(data, &ld); err != nil {
			break
		}
		ldJSON, jErr := json.MarshalIndent(ld, "", "\t")
		if jErr != nil {
			panic(jErr)
		}
		log.Infof("CreateLoop loop descriptor: %s", string(ldJSON))
		err = sn.handleCreateLoop(ctx, ld)

	case PacketCloseLoop:
		var ld routing.LoopData
		if err = json.Unmarshal(data, &ld); err != nil {
			break
		}
		err = sn.handleCloseLoop(ctx, ld.Loop.Remote.PubKey, routing.LoopData{
			Loop: routing.Loop{
				Remote: ld.Loop.Local,
				Local:  ld.Loop.Remote,
			},
		})

	default:
		err = errors.New("unknown foundation packet")
	}
	sn.metrics.Record(time.Since(startTime), err != nil)

	if err != nil {
		log.WithError(err).Warnf("Request completed with error.")
		return proto.WritePacket(RespFailure, err)
	}

	log.Infof("Request completed successfully.")
	return proto.WritePacket(RespSuccess, nil)
}

func (sn *Node) handleCreateLoop(ctx context.Context, ld routing.LoopDescriptor) error {
	src := ld.Loop.Local
	dst := ld.Loop.Remote

	// Reserve route IDs from visors.
	idr, err := sn.reserveRouteIDs(ctx, ld.Forward, ld.Reverse)
	if err != nil {
		return err
	}

	// Determine the rules to send to visors using loop descriptor and reserved route IDs.
	rulesMap, srcFwdRID, dstFwdRID, err := GenerateRules(idr, ld)
	if err != nil {
		return err
	}
	sn.Logger.Infof("generated rules: %v", rulesMap)

	// Add rules to visors.
	errCh := make(chan error, len(rulesMap))
	defer close(errCh)
	for pk, rules := range rulesMap {
		pk, rules := pk, rules
		go func() {
			log := sn.Logger.WithField("remote", pk)

			proto, err := sn.dialAndCreateProto(ctx, pk)
			if err != nil {
				log.WithError(err).Warn("failed to create proto")
				errCh <- err
				return
			}
			defer sn.closeProto(proto)
			log.Debug("proto created successfully")

			if err := AddRules(ctx, proto, rules); err != nil {
				log.WithError(err).Warn("failed to add rules")
				errCh <- err
				return
			}
			log.Debug("rules added")
			errCh <- nil
		}()
	}
	if err := finalError(len(rulesMap), errCh); err != nil {
		return err
	}

	// Confirm loop with responding visor.
	err = func() error {
		proto, err := sn.dialAndCreateProto(ctx, dst.PubKey)
		if err != nil {
			return err
		}
		defer sn.closeProto(proto)

		data := routing.LoopData{Loop: routing.Loop{Local: dst, Remote: src}, RouteID: dstFwdRID}
		return ConfirmLoop(ctx, proto, data)
	}()
	if err != nil {
		return fmt.Errorf("failed to confirm loop with destination visor: %v", err)
	}

	// Confirm loop with initiating visor.
	err = func() error {
		proto, err := sn.dialAndCreateProto(ctx, src.PubKey)
		if err != nil {
			return err
		}
		defer sn.closeProto(proto)

		data := routing.LoopData{Loop: routing.Loop{Local: src, Remote: dst}, RouteID: srcFwdRID}
		return ConfirmLoop(ctx, proto, data)
	}()
	if err != nil {
		return fmt.Errorf("failed to confirm loop with destination visor: %v", err)
	}

	return nil
}

func (sn *Node) reserveRouteIDs(ctx context.Context, fwd, rev routing.Route) (*idReservoir, error) {
	idc, total := newIDReservoir(fwd, rev)
	sn.Logger.Infof("There are %d route IDs to reserve.", total)

	err := idc.ReserveIDs(ctx, func(ctx context.Context, pk cipher.PubKey, n uint8) ([]routing.RouteID, error) {
		proto, err := sn.dialAndCreateProto(ctx, pk)
		if err != nil {
			return nil, err
		}
		defer sn.closeProto(proto)
		return RequestRouteIDs(ctx, proto, n)
	})
	if err != nil {
		sn.Logger.WithError(err).Warnf("Failed to reserve route IDs.")
		return nil, err
	}
	sn.Logger.Infof("Successfully reserved route IDs.")
	return idc, err
}

func (sn *Node) handleCloseLoop(ctx context.Context, on cipher.PubKey, ld routing.LoopData) error {
	proto, err := sn.dialAndCreateProto(ctx, on)
	if err != nil {
		return err
	}
	defer sn.closeProto(proto)

	if err := LoopClosed(ctx, proto, ld); err != nil {
		return err
	}

	sn.Logger.Infof("Closed loop on %s. LocalPort: %d", on, ld.Loop.Local.Port)
	return nil
}

func (sn *Node) dialAndCreateProto(ctx context.Context, pk cipher.PubKey) (*Protocol, error) {
	tr, err := sn.dmsgC.Dial(ctx, pk, snet.AwaitSetupPort)
	if err != nil {
		return nil, fmt.Errorf("transport: %s", err)
	}

	return NewSetupProtocol(tr), nil
}

func (sn *Node) closeProto(proto *Protocol) {
	if err := proto.Close(); err != nil {
		sn.Logger.Warn(err)
	}
}
