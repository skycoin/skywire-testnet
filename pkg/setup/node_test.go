// +build !no_ci

package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/SkycoinProject/dmsg"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/dmsg/disc"
	"github.com/SkycoinProject/skywire-mainnet/pkg/metrics"
	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
	"github.com/SkycoinProject/skywire-mainnet/pkg/snet"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"

	"github.com/SkycoinProject/skycoin/src/util/logging"
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
		logging.Disable()
	}

	os.Exit(m.Run())
}

// TODO(Darkren): fix this test. Explanation below
// Test may finish in 3 different ways:
// 1. Pass
// 2. Fail
// 3. Hang
// Adding `time.Sleep` at the start of `Write` operation in the DMSG makes it less possible to hang
// From observations seems like something's wrong in the DMSG, probably writing right after `Dial/Accept`
// causes this.
// 1. Test has possibility to pass, this means the test itself is correct
// 2. Test failure always comes with unexpected `context deadline exceeded`. In `read` operation of
// `setup proto` we ensure additional timeout, that's where this error comes from. This fact proves that
// DMSG has a related bug
// 3. Hanging may be not the problem of the DMSG. Probably some of the communication part here is wrong.
// The reason I think so is that - if we ensure read timeouts, why doesn't this test constantly fail?
// Maybe some wrapper for DMSG is wrong, or some internal operations before the actual communication behave bad
func TestNode(t *testing.T) {
	// Prepare mock dmsg discovery.
	discovery := disc.NewMock()

	// Prepare dmsg server.
	server, serverErr := createServer(t, discovery)
	defer func() {
		require.NoError(t, server.Close())
		require.NoError(t, errWithTimeout(serverErr))
	}()

	type clientWithDMSGAddrAndListener struct {
		*dmsg.Client
		Addr     dmsg.Addr
		Listener *dmsg.Listener
	}

	// CLOSURE: sets up dmsg clients.
	prepClients := func(n int) ([]clientWithDMSGAddrAndListener, func()) {
		clients := make([]clientWithDMSGAddrAndListener, n)
		for i := 0; i < n; i++ {
			var port uint16
			// setup node
			if i == 0 {
				port = snet.SetupPort
			} else {
				port = snet.AwaitSetupPort
			}
			pk, sk, err := cipher.GenerateDeterministicKeyPair([]byte{byte(i)})
			require.NoError(t, err)
			t.Logf("client[%d] PK: %s\n", i, pk)
			c := dmsg.NewClient(pk, sk, discovery, dmsg.SetLogger(logging.MustGetLogger(fmt.Sprintf("client_%d:%s:%d", i, pk, port))))
			require.NoError(t, c.InitiateServerConnections(context.TODO(), 1))
			listener, err := c.Listen(port)
			require.NoError(t, err)
			clients[i] = clientWithDMSGAddrAndListener{
				Client: c,
				Addr: dmsg.Addr{
					PK:   pk,
					Port: port,
				},
				Listener: listener,
			}
		}
		return clients, func() {
			for _, c := range clients {
				//require.NoError(t, c.Listener.Close())
				require.NoError(t, c.Close())
			}
		}
	}

	// CLOSURE: sets up setup node.
	prepSetupNode := func(c *dmsg.Client, listener *dmsg.Listener) (*Node, func()) {
		sn := &Node{
			Logger:  logging.MustGetLogger("setup_node"),
			dmsgC:   c,
			dmsgL:   listener,
			metrics: metrics.NewDummy(),
		}
		go func() {
			if err := sn.Serve(context.TODO()); err != nil {
				sn.Logger.WithError(err).Error("Failed to serve")
			}
		}()
		return sn, func() {
			require.NoError(t, sn.Close())
		}
	}

	//// TEST: Emulates the communication between 4 visor nodes and a setup node,
	//// where the first client node initiates a loop to the last.
	//t.Run("CreateLoop", func(t *testing.T) {
	//	// client index 0 is for setup node.
	//	// clients index 1 to 4 are for visor nodes.
	//	clients, closeClients := prepClients(5)
	//	defer closeClients()
	//
	//	// prepare and serve setup node (using client 0).
	//	_, closeSetup := prepSetupNode(clients[0].Client, clients[0].Listener)
	//	setupPK := clients[0].Addr.PK
	//	setupPort := clients[0].Addr.Port
	//	defer closeSetup()
	//
	//	// prepare loop creation (client_1 will use this to request loop creation with setup node).
	//	ld := routing.LoopDescriptor{
	//		Loop: routing.Loop{
	//			Local:  routing.Addr{PubKey: clients[1].Addr.PK, Port: 1},
	//			Remote: routing.Addr{PubKey: clients[4].Addr.PK, Port: 1},
	//		},
	//		Reverse: routing.Route{
	//			&routing.Hop{From: clients[1].Addr.PK, To: clients[2].Addr.PK, Transport: uuid.New()},
	//			&routing.Hop{From: clients[2].Addr.PK, To: clients[3].Addr.PK, Transport: uuid.New()},
	//			&routing.Hop{From: clients[3].Addr.PK, To: clients[4].Addr.PK, Transport: uuid.New()},
	//		},
	//		Forward: routing.Route{
	//			&routing.Hop{From: clients[4].Addr.PK, To: clients[3].Addr.PK, Transport: uuid.New()},
	//			&routing.Hop{From: clients[3].Addr.PK, To: clients[2].Addr.PK, Transport: uuid.New()},
	//			&routing.Hop{From: clients[2].Addr.PK, To: clients[1].Addr.PK, Transport: uuid.New()},
	//		},
	//		KeepAlive: 1 * time.Hour,
	//	}
	//
	//	// client_1 initiates loop creation with setup node.
	//	iTp, err := clients[1].Dial(context.TODO(), setupPK, setupPort)
	//	require.NoError(t, err)
	//	iTpErrs := make(chan error, 2)
	//	go func() {
	//		iTpErrs <- CreateLoop(context.TODO(), NewSetupProtocol(iTp), ld)
	//		iTpErrs <- iTp.Close()
	//		close(iTpErrs)
	//	}()
	//	defer func() {
	//		i := 0
	//		for err := range iTpErrs {
	//			require.NoError(t, err, i)
	//			i++
	//		}
	//	}()
	//
	//	var addRuleDone sync.WaitGroup
	//	var nextRouteID uint32
	//	// CLOSURE: emulates how a visor node should react when expecting an AddRules packet.
	//	expectAddRules := func(client int, expRule routing.RuleType) {
	//		conn, err := clients[client].Listener.Accept()
	//		require.NoError(t, err)
	//
	//		fmt.Printf("client %v:%v accepted\n", client, clients[client].Addr)
	//
	//		proto := NewSetupProtocol(conn)
	//
	//		pt, _, err := proto.ReadPacket()
	//		require.NoError(t, err)
	//		require.Equal(t, PacketRequestRouteID, pt)
	//
	//		fmt.Printf("client %v:%v got PacketRequestRouteID\n", client, clients[client].Addr)
	//
	//		routeID := atomic.AddUint32(&nextRouteID, 1)
	//
	//		// TODO: This error is not checked due to a bug in dmsg.
	//		_ = proto.WritePacket(RespSuccess, []routing.RouteID{routing.RouteID(routeID)}) // nolint:errcheck
	//		require.NoError(t, err)
	//
	//		fmt.Printf("client %v:%v responded to with registration ID: %v\n", client, clients[client].Addr, routeID)
	//
	//		require.NoError(t, conn.Close())
	//
	//		conn, err = clients[client].Listener.Accept()
	//		require.NoError(t, err)
	//
	//		fmt.Printf("client %v:%v accepted 2nd time\n", client, clients[client].Addr)
	//
	//		proto = NewSetupProtocol(conn)
	//
	//		pt, pp, err := proto.ReadPacket()
	//		require.NoError(t, err)
	//		require.Equal(t, PacketAddRules, pt)
	//
	//		fmt.Printf("client %v:%v got PacketAddRules\n", client, clients[client].Addr)
	//
	//		var rs []routing.Rule
	//		require.NoError(t, json.Unmarshal(pp, &rs))
	//
	//		for _, r := range rs {
	//			require.Equal(t, expRule, r.Type())
	//		}
	//
	//		// TODO: This error is not checked due to a bug in dmsg.
	//		err = proto.WritePacket(RespSuccess, nil)
	//		_ = err
	//
	//		fmt.Printf("client %v:%v responded for PacketAddRules\n", client, clients[client].Addr)
	//
	//		require.NoError(t, conn.Close())
	//
	//		addRuleDone.Done()
	//	}
	//
	//	// CLOSURE: emulates how a visor node should react when expecting an OnConfirmLoop packet.
	//	expectConfirmLoop := func(client int) {
	//		tp, err := clients[client].Listener.AcceptTransport()
	//		require.NoError(t, err)
	//
	//		proto := NewSetupProtocol(tp)
	//
	//		pt, pp, err := proto.ReadPacket()
	//		require.NoError(t, err)
	//		require.Equal(t, PacketConfirmLoop, pt)
	//
	//		var d routing.LoopData
	//		require.NoError(t, json.Unmarshal(pp, &d))
	//
	//		switch client {
	//		case 1:
	//			require.Equal(t, ld.Loop, d.Loop)
	//		case 4:
	//			require.Equal(t, ld.Loop.Local, d.Loop.Remote)
	//			require.Equal(t, ld.Loop.Remote, d.Loop.Local)
	//		default:
	//			t.Fatalf("We shouldn't be receiving a OnConfirmLoop packet from client %d", client)
	//		}
	//
	//		// TODO: This error is not checked due to a bug in dmsg.
	//		err = proto.WritePacket(RespSuccess, nil)
	//		_ = err
	//
	//		require.NoError(t, tp.Close())
	//	}
	//
	//	// since the route establishment is asynchronous,
	//	// we must expect all the messages in parallel
	//	addRuleDone.Add(4)
	//	go expectAddRules(4, routing.RuleApp)
	//	go expectAddRules(3, routing.RuleForward)
	//	go expectAddRules(2, routing.RuleForward)
	//	go expectAddRules(1, routing.RuleForward)
	//	addRuleDone.Wait()
	//	fmt.Println("FORWARD ROUTE DONE")
	//	addRuleDone.Add(4)
	//	go expectAddRules(1, routing.RuleApp)
	//	go expectAddRules(2, routing.RuleForward)
	//	go expectAddRules(3, routing.RuleForward)
	//	go expectAddRules(4, routing.RuleForward)
	//	addRuleDone.Wait()
	//	fmt.Println("REVERSE ROUTE DONE")
	//	expectConfirmLoop(1)
	//	expectConfirmLoop(4)
	//})

	// TEST: Emulates the communication between 2 visor nodes and a setup nodes,
	// where a route is already established,
	// and the first client attempts to tear it down.
	t.Run("CloseLoop", func(t *testing.T) {
		// client index 0 is for setup node.
		// clients index 1 and 2 are for visor nodes.
		clients, closeClients := prepClients(3)
		defer closeClients()

		// prepare and serve setup node.
		_, closeSetup := prepSetupNode(clients[0].Client, clients[0].Listener)
		setupPK := clients[0].Addr.PK
		setupPort := clients[0].Addr.Port
		defer closeSetup()

		// prepare loop data describing the loop that is to be closed.
		ld := routing.LoopData{
			Loop: routing.Loop{
				Local: routing.Addr{
					PubKey: clients[1].Addr.PK,
					Port:   1,
				},
				Remote: routing.Addr{
					PubKey: clients[2].Addr.PK,
					Port:   2,
				},
			},
			RouteID: 3,
		}

		// client_1 initiates close loop with setup node.
		iTp, err := clients[1].Dial(context.TODO(), setupPK, setupPort)
		require.NoError(t, err)
		iTpErrs := make(chan error, 2)
		go func() {
			iTpErrs <- CloseLoop(context.TODO(), NewSetupProtocol(iTp), ld)
			iTpErrs <- iTp.Close()
			close(iTpErrs)
		}()
		defer func() {
			i := 0
			for err := range iTpErrs {
				require.NoError(t, err, i)
				i++
			}
		}()

		// client_2 accepts close request.
		tp, err := clients[2].Listener.AcceptTransport()
		require.NoError(t, err)
		defer func() { require.NoError(t, tp.Close()) }()

		proto := NewSetupProtocol(tp)

		pt, pp, err := proto.ReadPacket()
		require.NoError(t, err)
		require.Equal(t, PacketLoopClosed, pt)

		var d routing.LoopData
		require.NoError(t, json.Unmarshal(pp, &d))
		require.Equal(t, ld.Loop.Remote, d.Loop.Local)
		require.Equal(t, ld.Loop.Local, d.Loop.Remote)

		// TODO: This error is not checked due to a bug in dmsg.
		err = proto.WritePacket(RespSuccess, nil)
		_ = err
	})
}

func createServer(t *testing.T, dc disc.APIClient) (srv *dmsg.Server, srvErr <-chan error) {
	pk, sk, err := cipher.GenerateDeterministicKeyPair([]byte("s"))
	require.NoError(t, err)
	l, err := nettest.NewLocalListener("tcp")
	require.NoError(t, err)
	srv, err = dmsg.NewServer(pk, sk, "", l, dc)
	require.NoError(t, err)
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
		close(errCh)
	}()
	return srv, errCh
}

func errWithTimeout(ch <-chan error) error {
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		return errors.New("timeout")
	}
}
