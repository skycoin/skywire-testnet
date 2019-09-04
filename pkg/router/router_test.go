package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/skycoin/dmsg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/app"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/snet"
	"github.com/skycoin/skywire/pkg/snet/snettest"
	"github.com/skycoin/skywire/pkg/transport"

	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/logging"
)

func TestMain(m *testing.M) {
	loggingLevel, ok := os.LookupEnv("TEST_LOGGING_LEVEL")
	if ok {
		lvl, err := logging.LevelFromString(loggingLevel)
		if err != nil {
			log.Fatal(err)
		}
		logging.SetLevel(lvl)
	} else {
		logging.SetLevel(logrus.TraceLevel)
	}
	os.Exit(m.Run())
}

// Ensure that received packets are handled properly in `(*Router).Serve()`.
func TestRouter_Serve(t *testing.T) {
	// We are generating two key pairs - one for the a `Router`, the other to send packets to `Router`.
	keys := snettest.GenKeyPairs(2)

	// create test env
	nEnv := snettest.NewEnv(t, keys)
	defer nEnv.Teardown()
	rEnv := NewTestEnv(t, nEnv.Nets)
	defer rEnv.Teardown()

	// Create routers
	r0, err := New(nEnv.Nets[0], rEnv.GenRouterConfig(0))
	require.NoError(t, err)
	// go r0.Serve(context.TODO())
	r1, err := New(nEnv.Nets[1], rEnv.GenRouterConfig(1))
	require.NoError(t, err)
	// go r1.Serve(context.TODO())

	// Create dmsg transport between two `snet.Network` entities.
	tp1, err := rEnv.TpMngrs[1].SaveTransport(context.TODO(), keys[0].PK, dmsg.Type)
	require.NoError(t, err)

	// CLOSURE: clear all rules in all router.
	clearRules := func(routers ...*Router) {
		for _, r := range routers {
			var rtIDs []routing.RouteID
			require.NoError(t, r.rm.rt.RangeRules(func(rtID routing.RouteID, _ routing.Rule) bool {
				rtIDs = append(rtIDs, rtID)
				return true
			}))
			require.NoError(t, r.rm.rt.DeleteRules(rtIDs...))
		}
	}

	// TEST: Ensure handlePacket does as expected.
	// After setting a rule in r0, r0 should forward a packet to r1 (as specified in the given rule)
	// when r0.handlePacket() is called.
	t.Run("handlePacket_fwdRule", func(t *testing.T) {
		defer clearRules(r0, r1)

		// Add a FWD rule for r0.
		fwdRule := routing.ForwardRule(1*time.Hour, routing.RouteID(5), tp1.Entry.ID, routing.RouteID(0))
		fwdRtID, err := r0.rm.rt.AddRule(fwdRule)
		require.NoError(t, err)

		// Call handlePacket for r0 (this should in turn, use the rule we added).
		packet := routing.MakePacket(fwdRtID, []byte("This is a test!"))
		require.NoError(t, r0.handlePacket(context.TODO(), packet))

		// r1 should receive the packet handled by r0.
		recvPacket, err := r1.tm.ReadPacket()
		assert.NoError(t, err)
		assert.Equal(t, packet.Size(), recvPacket.Size())
		assert.Equal(t, packet.Payload(), recvPacket.Payload())
		assert.Equal(t, fwdRtID, packet.RouteID())
	})

	// TODO(evanlinjin): I'm having so much trouble with this I officially give up.
	t.Run("handlePacket_appRule", func(t *testing.T) {
		const duration = 10 * time.Second
		// time.AfterFunc(duration, func() {
		// 	panic("timeout")
		// })

		defer clearRules(r0, r1)

		// prepare mock-app
		localPort := routing.Port(9)
		cConn, sConn := net.Pipe()

		// mock-app config
		appConf := &app.Config{
			AppName:         "test_app",
			AppVersion:      "1.0",
			ProtocolVersion: supportedProtocolVersion,
		}

		// serve mock-app
		// sErrCh := make(chan error, 1)
		go func() {
			// sErrCh <- r0.ServeApp(sConn, localPort, appConf)
			_ = r0.ServeApp(sConn, localPort, appConf)
			// close(sErrCh)
		}()
		// defer func() {
		// 	assert.NoError(t, cConn.Close())
		// 	assert.NoError(t, <-sErrCh)
		// }()

		a, err := app.New(cConn, appConf)
		require.NoError(t, err)
		// cErrCh := make(chan error, 1)
		go func() {
			conn, err := a.Accept()
			if err == nil {
				fmt.Println("ACCEPTED:", conn.RemoteAddr())
			}
			fmt.Println("FAILED TO ACCEPT")
			// cErrCh <- err
			// close(cErrCh)
		}()
		a.Dial(a.Addr().(routing.Addr))
		// defer func() {
		// 	assert.NoError(t, <-cErrCh)
		// }()

		// Add a APP rule for r0.
		// port8 := routing.Port(8)
		appRule := routing.AppRule(10*time.Minute, 0, routing.RouteID(7), keys[1].PK, localPort, localPort)
		appRtID, err := r0.rm.rt.AddRule(appRule)
		require.NoError(t, err)

		// Call handlePacket for r0.

		// payload is prepended with two bytes to satisfy app.Proto.
		// payload[0] = frame type, payload[1] = id
		rAddr := routing.Addr{PubKey: keys[1].PK, Port: localPort}
		rawRAddr, _ := json.Marshal(rAddr)
		// payload := append([]byte{byte(app.FrameClose), 0}, rawRAddr...)
		packet := routing.MakePacket(appRtID, rawRAddr)
		require.NoError(t, r0.handlePacket(context.TODO(), packet))
	})
}

type TestEnv struct {
	TpD transport.DiscoveryClient

	TpMngrConfs []*transport.ManagerConfig
	TpMngrs     []*transport.Manager

	teardown func()
}

func NewTestEnv(t *testing.T, nets []*snet.Network) *TestEnv {
	tpD := transport.NewDiscoveryMock()

	mConfs := make([]*transport.ManagerConfig, len(nets))
	ms := make([]*transport.Manager, len(nets))

	for i, n := range nets {
		var err error

		mConfs[i] = &transport.ManagerConfig{
			PubKey:          n.LocalPK(),
			SecKey:          n.LocalSK(),
			DiscoveryClient: tpD,
			LogStore:        transport.InMemoryTransportLogStore(),
		}

		ms[i], err = transport.NewManager(n, mConfs[i])
		require.NoError(t, err)

		go ms[i].Serve(context.TODO())
	}

	teardown := func() {
		for _, m := range ms {
			assert.NoError(t, m.Close())
		}
	}

	return &TestEnv{
		TpD:         tpD,
		TpMngrConfs: mConfs,
		TpMngrs:     ms,
		teardown:    teardown,
	}
}

func (e *TestEnv) GenRouterConfig(i int) *Config {
	return &Config{
		Logger:                 logging.MustGetLogger(fmt.Sprintf("router_%d", i)),
		PubKey:                 e.TpMngrConfs[i].PubKey,
		SecKey:                 e.TpMngrConfs[i].SecKey,
		TransportManager:       e.TpMngrs[i],
		RoutingTable:           routing.InMemoryRoutingTable(),
		RouteFinder:            routeFinder.NewMock(),
		SetupNodes:             nil, // TODO
		GarbageCollectDuration: DefaultGarbageCollectDuration,
	}
}

func (e *TestEnv) Teardown() {
	e.teardown()
}
