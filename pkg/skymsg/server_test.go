package skymsg

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/transport"
)

// Ensure Server starts and quits with no error.
func TestNewServer(t *testing.T) {
	sPK, sSK := cipher.GenerateKeyPair()
	sAddr := ":8080"

	//aPK, aSK := cipher.GenerateKeyPair()
	//bPK, bSK := cipher.GenerateKeyPair()

	dc := client.NewMock()

	s := NewServer(sPK, sSK, sAddr, dc)
	go s.ListenAndServe(sAddr)

	time.Sleep(time.Second)

	assert.NoError(t, s.Close())
}

func TestNewClient(t *testing.T) {
	aPK, aSK := cipher.GenerateKeyPair()
	bPK, bSK := cipher.GenerateKeyPair()
	sPK, sSK := cipher.GenerateKeyPair()
	sAddr := ":8080"

	dc := client.NewMock()

	s := NewServer(sPK, sSK, sAddr, dc)
	go s.ListenAndServe(sAddr)

	a := NewClient(aPK, aSK, dc)
	require.NoError(t, a.InitiateLinks(context.Background(), 1))
	aDone := make(chan struct{})
	var aErr error
	var aTp transport.Transport
	go func() {
		aTp, aErr = a.Accept(context.Background())
		close(aDone)
	}()

	b := NewClient(bPK, bSK, dc)
	require.NoError(t, a.InitiateLinks(context.TODO(), 1))
	bTp, err := b.Dial(context.TODO(), aPK)
	require.NoError(t, err)

	<-aDone
	require.NoError(t, aErr)

	for i := 0; i < 10; i++ {
		pay := []byte(fmt.Sprintf("This is message %d!", i))

		n, err := aTp.Write(pay)
		require.Equal(t, len(pay), n)
		require.NoError(t, err)

		got := make([]byte, len(pay))
		n, err = bTp.Read(got)
		require.Equal(t, len(pay), n)
		require.NoError(t, err)
		require.Equal(t, pay, got)
	}

	// Close TPs
	require.NoError(t, aTp.Close())
	require.NoError(t, bTp.Close())

	// Close server.
	assert.NoError(t, s.Close())
}
