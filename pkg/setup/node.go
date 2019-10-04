package setup

import (
	"context"
	"fmt"
	"net/rpc"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/disc"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/metrics"
	"github.com/skycoin/skywire/pkg/snet"
)

// Node performs routes setup operations over messaging channel.
type Node struct {
	logger   *logging.Logger
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

	node := &Node{
		logger:   log,
		dmsgC:    dmsgC,
		dmsgL:    dmsgL,
		srvCount: conf.Messaging.ServerCount,
		metrics:  metrics,
	}

	return node, nil
}

// Close closes underlying dmsg client.
func (sn *Node) Close() error {
	if sn == nil {
		return nil
	}

	return sn.dmsgC.Close()
}

// Serve starts transport listening loop.
func (sn *Node) Serve() error {
	sn.logger.Info("Serving setup node")

	for {
		conn, err := sn.dmsgL.AcceptTransport()
		if err != nil {
			return err
		}

		sn.logger.WithField("requester", conn.RemotePK()).Infof("Received request.")

		rpcS := rpc.NewServer()
		if err := rpcS.Register(NewRPCGateway(conn.RemotePK(), sn)); err != nil {
			return err
		}
		go rpcS.ServeConn(conn)
	}
}
