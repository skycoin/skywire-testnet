package snettest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

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
	Keys     []KeyPair
	Nets     []*snet.Network
	teardown func()
}

// NewEnv creates a `network.Network` test environment.
// `nPairs` is the public/private key pairs of all the `network.Network`s to be created.
func NewEnv(t *testing.T, keys []KeyPair) *Env {

	// Prepare `dmsg`.
	dmsgD := disc.NewMock()
	dmsgS, dmsgSErr := createDmsgSrv(t, dmsgD)

	// Prepare `snets`.
	ns := make([]*snet.Network, len(keys))
	for i, pairs := range keys {
		n := snet.NewRaw(
			snet.Config{
				PubKey:      pairs.PK,
				SecKey:      pairs.SK,
				TpNetworks:  []string{dmsg.Type},
				DmsgMinSrvs: 1,
			},
			dmsg.NewClient(pairs.PK, pairs.SK, dmsgD),
		)
		require.NoError(t, n.Init(context.TODO()))
		ns[i] = n
	}

	// Prepare teardown closure.
	teardown := func() {
		for _, n := range ns {
			assert.NoError(t, n.Close())
		}
		assert.NoError(t, dmsgS.Close())
		for err := range dmsgSErr {
			assert.NoError(t, err)
		}
	}

	return &Env{
		DmsgD:    dmsgD,
		DmsgS:    dmsgS,
		Keys:     keys,
		Nets:     ns,
		teardown: teardown,
	}
}

// Teardown shutdowns the Env.
func (e *Env) Teardown() { e.teardown() }

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
