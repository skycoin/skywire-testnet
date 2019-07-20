package setup

import (
	"log"
	"os"
	"testing"

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
		logging.Disable()
	}

	os.Exit(m.Run())
}

//func TestCreateLoop(t *testing.T) {
//	dc := disc.NewMock()
//
//	pk1, sk1 := cipher.GenerateKeyPair()
//	pk2, _ := cipher.GenerateKeyPair()
//	pk3, sk3 := cipher.GenerateKeyPair()
//	pk4, sk4 := cipher.GenerateKeyPair()
//	pkS, _ := cipher.GenerateKeyPair()
//
//	n1, srvErrCh1, err := createServer(dc)
//	require.NoError(t, err)
//
//	n2, srvErrCh2, err := createServer(dc)
//	require.NoError(t, err)
//
//	n3, srvErrCh3, err := createServer(dc)
//	require.NoError(t, err)
//
//	c1 := dmsg.NewClient(pk1, sk1, dc)
//	// c2 := dmsg.NewClient(pk2, sk2, dc)
//	c3 := dmsg.NewClient(pk3, sk3, dc)
//	c4 := dmsg.NewClient(pk4, sk4, dc)
//
//	_, err = c1.Dial(context.TODO(), pk2)
//	require.NoError(t, err)
//
//	_, err = c3.Dial(context.TODO(), pk4)
//	require.NoError(t, err)
//
//	lPK, _ := cipher.GenerateKeyPair()
//	rPK, _ := cipher.GenerateKeyPair()
//	ld := routing.LoopDescriptor{Loop: routing.Loop{Local: routing.Addr{PubKey: lPK, Port: 1}, Remote: routing.Addr{PubKey: rPK, Port: 2}}, Expiry: time.Now().Add(time.Hour),
//		Forward: routing.Route{
//			&routing.Hop{From: pk1, To: pk2, Transport: uuid.New()},
//			&routing.Hop{From: pk2, To: pk3, Transport: uuid.New()},
//		},
//		Reverse: routing.Route{
//			&routing.Hop{From: pk3, To: pk2, Transport: uuid.New()},
//			&routing.Hop{From: pk2, To: pk1, Transport: uuid.New()},
//		},
//	}
//
//	time.Sleep(100 * time.Millisecond)
//
//	sn := &Node{logging.MustGetLogger("routesetup"), c1, 0, metrics.NewDummy()}
//	errChan := make(chan error)
//	go func() {
//		errChan <- sn.Serve(context.TODO())
//	}()
//
//	tr, err := c4.Dial(context.TODO(), pkS)
//	require.NoError(t, err)
//
//	proto := NewSetupProtocol(tr)
//	require.NoError(t, CreateLoop(proto, ld))
//
//	require.NoError(t, sn.Close())
//	require.NoError(t, <-errChan)
//
//	require.NoError(t, n1.Close())
//	require.NoError(t, errWithTimeout(srvErrCh1))
//
//	require.NoError(t, n2.Close())
//	require.NoError(t, errWithTimeout(srvErrCh2))
//
//	require.NoError(t, n3.Close())
//	require.NoError(t, errWithTimeout(srvErrCh3))
//}

//func TestCloseLoop(t *testing.T) {
//	dc := disc.NewMock()
//
//	pk1, sk1 := cipher.GenerateKeyPair()
//	pk3, sk3 := cipher.GenerateKeyPair()
//
//	n3, srvErrCh, err := createServer(dc)
//	require.NoError(t, err)
//
//	time.Sleep(100 * time.Millisecond)
//
//	c1 := dmsg.NewClient(pk1, sk1, dc)
//	c3 := dmsg.NewClient(pk3, sk3, dc)
//
//	require.NoError(t, c1.InitiateServerConnections(context.Background(), 1))
//	require.NoError(t, c3.InitiateServerConnections(context.Background(), 1))
//
//	sn := &Node{logging.MustGetLogger("routesetup"), c3, 0, metrics.NewDummy()}
//	errChan := make(chan error)
//	go func() {
//		errChan <- sn.Serve(context.TODO())
//	}()
//
//	tr, err := c1.Dial(context.TODO(), pk3)
//	require.NoError(t, err)
//
//	proto := NewSetupProtocol(tr)
//	require.NoError(t, CloseLoop(proto, routing.LoopData{
//		Loop: routing.Loop{
//			Remote: routing.Addr{
//				PubKey: pk3,
//				Port:   2,
//			},
//			Local: routing.Addr{
//				Port: 1,
//			},
//		},
//	}))
//
//	require.NoError(t, sn.Close())
//	require.NoError(t, <-errChan)
//
//	require.NoError(t, n3.Close())
//	require.NoError(t, errWithTimeout(srvErrCh))
//}

//func createServer(dc disc.APIClient) (srv *dmsg.Server, srvErr <-chan error, err error) {
//	pk, sk := cipher.GenerateKeyPair()
//
//	l, err := nettest.NewLocalListener("tcp")
//	if err != nil {
//		return nil, nil, err
//	}
//
//	srv, err = dmsg.NewServer(pk, sk, "", l, dc)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	errCh := make(chan error, 1)
//	go func() {
//		errCh <- srv.Serve()
//	}()
//
//	return srv, errCh, nil
//}
//
//func errWithTimeout(ch <-chan error) error {
//	select {
//	case err := <-ch:
//		return err
//	case <-time.After(5 * time.Second):
//		return errors.New("timeout")
//	}
//}
