package testsnet

import (
	"testing"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"

	"github.com/skycoin/skywire/pkg/snet"
)

// KeyPair holds a public/private key pair.
type KeyPair struct {
	PK cipher.PubKey
	SK cipher.SecKey
}

// GenKeyPairs generates 'n' number of key pairs.
func GenKeyPairs(n int) []KeyPair {
	pairs := make([]KeyPair, n)
	for i := range pairs {
		pk, sk, err := cipher.GenerateDeterministicKeyPair([]byte{byte(i)})
		if err != nil {
			panic(err)
		}
		pairs[i] = KeyPair{PK: pk, SK: sk}
	}
	return pairs
}

// Env contains a network test environment.
type Env struct {
	DmsgD    disc.APIClient
	DmsgS    *dmsg.Server
	Nets     []*snet.Network
	teardown func()
}

// NewEnv creates a `network.Network` test environment.
// `nPairs` is the public/private key pairs of all the `network.Network`s to be created.
func NewEnv(t *testing.T, nPairs []KeyPair) *Env {
	return nil

	//// Prepare `dmsg`.
	//dmsgD := disc.NewMock()
	//dmsgS, dmsgSErr := createDmsgSrv(t, dmsgD)
	//
	//ns := make([]*snet.Network, len(nPairs))
	//for i, pairs := range nPairs {
	//
	//}
	//
	//// Prepare teardown closure.
	//teardown := func() {
	//	require.NoError(t, <-dmsgSErr)
	//}

}

//// SetupTestEnv creates a dmsg TestEnv.
//func SetupTestEnv(t *testing.T, keyPairs []KeyPair) *TestEnv {
//	discovery := disc.NewMock()
//
//	srv, srvErr := createServer(t, discovery)
//
//	clients := make([]*Client, len(keyPairs))
//	for i, pair := range keyPairs {
//		t.Logf("dmsg_client[%d] PK: %s\n", i, pair.PK)
//		c := NewClient(pair.PK, pair.SK, discovery,
//			SetLogger(logging.MustGetLogger(fmt.Sprintf("client_%d:%s", i, pair.PK.String()[:6]))))
//		require.NoError(t, c.InitiateServerConnections(context.TODO(), 1))
//		clients[i] = c
//	}
//
//	teardown := func() {
//		for _, c := range clients {
//			require.NoError(t, c.Close())
//		}
//		require.NoError(t, srv.Close())
//		for err := range srvErr {
//			require.NoError(t, err)
//		}
//	}
//
//	return &TestEnv{
//		Disc:     discovery,
//		Srv:      srv,
//		Clients:  clients,
//		teardown: teardown,
//	}
//}
//
//// TearDown shutdowns the TestEnv.
//func (e *TestEnv) TearDown() { e.teardown() }
//
func createDmsgSrv(t *testing.T, dc disc.APIClient) (srv *dmsg.Server, srvErr <-chan error) {
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
