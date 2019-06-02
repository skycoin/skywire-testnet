package dmsg

import (
	"context"
	"fmt"
	"golang.org/x/net/nettest"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/transport"
)

// Ensure Server starts and quits with no error.
func TestNewServer(t *testing.T) {
	sPK, sSK := cipher.GenerateKeyPair()
	dc := client.NewMock()

	s := NewServer(sPK, sSK, "", dc)
	go s.ListenAndServe("") //nolint:errcheck

	time.Sleep(time.Second)

	assert.NoError(t, s.Close())
}

// Given two client instances (a & b) and a server instance (s),
// Client b should be able to dial a transport with client b
// Data should be sent and delivered successfully via the transport.
// TODO: fix this.
func TestNewClient(t *testing.T) {
	aPK, aSK := cipher.GenerateKeyPair()
	bPK, bSK := cipher.GenerateKeyPair()
	sPK, sSK := cipher.GenerateKeyPair()
	sAddr := ":8081"

	const tpCount = 10
	const msgCount = 100

	dc := client.NewMock()

	s := NewServer(sPK, sSK, sAddr, dc)
	go s.ListenAndServe(sAddr) //nolint:errcheck

	a := NewClient(aPK, aSK, dc)
	a.SetLogger(logging.MustGetLogger("A"))
	require.NoError(t, a.InitiateServerConnections(context.Background(), 1))
	nettest.NewLocalListener()

	b := NewClient(bPK, bSK, dc)
	b.SetLogger(logging.MustGetLogger("B"))
	require.NoError(t, b.InitiateServerConnections(context.Background(), 1))

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < tpCount; i++ {
			aDone := make(chan struct{})
			var aTp transport.Transport
			go func() {
				var err error
				aTp, err = a.Accept(context.Background())
				catch(err)
				close(aDone)
			}()

			bTp, err := b.Dial(context.Background(), aPK)
			catch(err)

			<-aDone
			catch(err)

			for j := 0; j < msgCount; j++ {
				pay := []byte(fmt.Sprintf("This is message %d!", j))
				_, err := aTp.Write(pay)
				catch(err)
				_, err = bTp.Read(pay)
				catch(err)
			}

			// Close TPs
			catch(aTp.Close())
			catch(bTp.Close())
		}
	}()

	for i := 0; i < tpCount; i++ {
		bDone := make(chan struct{})
		var bErr error
		var bTp transport.Transport
		go func() {
			bTp, bErr = b.Accept(context.Background())
			close(bDone)
		}()

		aTp, err := a.Dial(context.Background(), bPK)
		require.NoError(t, err)

		<-bDone
		require.NoError(t, bErr)

		for j := 0; j < msgCount; j++ {
			pay := []byte(fmt.Sprintf("This is message %d!", j))

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
	}
	wg.Wait()

	// Close server.
	assert.NoError(t, s.Close())
}

func catch(err error) {
	if err != nil {
		panic(err)
	}
}

func TestNewConn(t *testing.T) {

}
