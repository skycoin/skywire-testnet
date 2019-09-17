package dmsg

import (
	"context"
	"fmt"
	"testing"

	"github.com/SkycoinProject/dmsg"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/dmsg/disc"
	"github.com/SkycoinProject/skycoin/src/util/logging"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"
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

// TestEnv contains a dmsg environment.
type TestEnv struct {
	Disc     disc.APIClient
	Srv      *Server
	Clients  []*Client
	teardown func()
}

// SetupTestEnv creates a dmsg TestEnv.
func SetupTestEnv(t *testing.T, keyPairs []KeyPair) *TestEnv {
	discovery := disc.NewMock()

	srv, srvErr := createServer(t, discovery)

	clients := make([]*Client, len(keyPairs))
	for i, pair := range keyPairs {
		t.Logf("dmsg_client[%d] PK: %s\n", i, pair.PK)
		c := NewClient(pair.PK, pair.SK, discovery,
			SetLogger(logging.MustGetLogger(fmt.Sprintf("client_%d:%s", i, pair.PK.String()[:6]))))
		require.NoError(t, c.InitiateServerConnections(context.TODO(), 1))
		clients[i] = c
	}

	teardown := func() {
		for _, c := range clients {
			require.NoError(t, c.Close())
		}
		require.NoError(t, srv.Close())
		for err := range srvErr {
			require.NoError(t, err)
		}
	}

	return &TestEnv{
		Disc:     discovery,
		Srv:      srv,
		Clients:  clients,
		teardown: teardown,
	}
}

// TearDown shutdowns the TestEnv.
func (e *TestEnv) TearDown() { e.teardown() }

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
