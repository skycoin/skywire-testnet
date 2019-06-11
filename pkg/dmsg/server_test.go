package dmsg

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/nettest"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/transport"
)

func TestServerConn_AddNext(t *testing.T) {
	type want struct {
		id      uint16
		wantErr bool
	}

	pk, _ := cipher.GenerateKeyPair()

	var fullNextConns [math.MaxUint16 + 1]*NextConn
	fullNextConns[1] = &NextConn{}
	for i := uint16(3); i != 1; i += 2 {
		fullNextConns[i] = &NextConn{}
	}

	timeoutCtx, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)

	cases := []struct {
		name       string
		conn       *ServerConn
		ctx        context.Context
		nextConnID uint16
		want       want
	}{
		{
			name: "ok",
			conn: &ServerConn{
				remoteClient: pk,
				log:          logging.MustGetLogger("ServerConn"),
				nextRespID:   1,
			},
			ctx: context.Background(),
			want: want{
				id: 1,
			},
		},
		{
			name: "ok, skip 1",
			conn: &ServerConn{
				remoteClient: pk,
				log:          logging.MustGetLogger("ServerConn"),
				nextRespID:   1,
				nextConns: [math.MaxUint16 + 1]*NextConn{
					1: {},
				},
			},
			ctx: context.Background(),
			want: want{
				id: 3,
			},
		},
		{
			name: "fail - timed out",
			conn: &ServerConn{
				remoteClient: pk,
				log:          logging.MustGetLogger("ServerConn"),
				nextRespID:   1,
				nextConns:    fullNextConns,
			},
			ctx: timeoutCtx,
			want: want{
				wantErr: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := tc.conn.addNext(tc.ctx, &NextConn{})

			if tc.want.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if err != nil {
				return
			}

			require.Equal(t, tc.want.id, id)
		})
	}

	// concurrent
	connsCount := 50

	serverConn := &ServerConn{
		log:          logging.MustGetLogger("ServerConn"),
		remoteClient: pk,
		nextRespID:   1,
	}

	var wg sync.WaitGroup
	wg.Add(connsCount)
	for i := 0; i < connsCount; i++ {
		go func() {
			_, err := serverConn.addNext(context.Background(), &NextConn{})
			require.NoError(t, err)

			wg.Done()
		}()
	}

	wg.Wait()

	for i := uint16(1); i < uint16(connsCount*2); i += 2 {
		require.NotNil(t, serverConn.nextConns[i])
	}

	for i := uint16(connsCount*2 + 1); i != 1; i += 2 {
		require.Nil(t, serverConn.nextConns[i])
	}
}

// Ensure Server starts and quits with no error.
func TestNewServer(t *testing.T) {
	sPK, sSK := cipher.GenerateKeyPair()
	dc := client.NewMock()

	l, err := net.Listen("tcp", "")
	require.NoError(t, err)

	s, err := NewServer(sPK, sSK, l, dc)
	require.NoError(t, err)

	go s.Serve() //nolint:errcheck

	time.Sleep(time.Second)

	assert.NoError(t, s.Close())
}

func TestServer_Serve(t *testing.T) {
	sPK, sSK := cipher.GenerateKeyPair()
	dc := client.NewMock()

	l, err := nettest.NewLocalListener("tcp")
	require.NoError(t, err)

	s, err := NewServer(sPK, sSK, l, dc)
	require.NoError(t, err)

	go s.Serve() //nolint:errcheck

	// connect two clients, establish transport, check if there are
	// two ServerConn's and that both conn's `nextConn` is filled correctly
	t.Run("test transport establishment", func(t *testing.T) {
		aPK, aSK := cipher.GenerateKeyPair()
		bPK, bSK := cipher.GenerateKeyPair()

		a := NewClient(aPK, aSK, dc)
		a.SetLogger(logging.MustGetLogger("A"))
		err := a.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		b := NewClient(bPK, bSK, dc)
		b.SetLogger(logging.MustGetLogger("B"))
		err = b.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		aDone := make(chan struct{})
		var aTransport transport.Transport
		var aErr error
		go func() {
			aTransport, aErr = a.Accept(context.Background())

			close(aDone)
		}()

		bTransport, err := b.Dial(context.Background(), aPK)
		require.NoError(t, err)

		<-aDone
		require.NoError(t, aErr)

		// must be 2 ServerConn's
		require.Equal(t, 2, len(s.conns))

		// must have ServerConn for A
		aServerConn, ok := s.conns[aPK]
		require.Equal(t, true, ok)
		require.Equal(t, aPK, aServerConn.remoteClient)

		// must have ServerConn for B
		bServerConn, ok := s.conns[bPK]
		require.Equal(t, true, ok)
		require.Equal(t, bPK, bServerConn.remoteClient)

		// must have a ClientConn
		aClientConn, ok := a.conns[sPK]
		require.Equal(t, true, ok)
		require.Equal(t, sPK, aClientConn.remoteSrv)

		// must have a ClientConn
		bClientConn, ok := b.conns[sPK]
		require.Equal(t, true, ok)
		require.Equal(t, sPK, bClientConn.remoteSrv)

		// check whether nextConn's contents are as must be
		bNextConn := bServerConn.nextConns[bClientConn.nextInitID-2]
		require.NotNil(t, bNextConn)
		require.Equal(t, bNextConn.id, aServerConn.nextRespID-2)

		// check whether nextConn's contents are as must be
		aNextConn := aServerConn.nextConns[aServerConn.nextRespID-2]
		require.NotNil(t, aNextConn)
		require.Equal(t, aNextConn.id, bClientConn.nextInitID-2)

		err = aTransport.Close()
		require.NoError(t, err)

		err = bTransport.Close()
		require.NoError(t, err)

		err = a.Close()
		require.NoError(t, err)

		err = b.Close()
		require.NoError(t, err)

		require.NoError(t, testWithTimeout(t, 5*time.Second, func() error {
			s.mx.Lock()
			l := len(s.conns)
			s.mx.Unlock()

			if l != 0 {
				return errors.New("s.conns is not empty")
			}

			return nil
		}))

		require.NoError(t, testWithTimeout(t, 5*time.Second, func() error {
			a.mx.Lock()
			l := len(a.conns)
			a.mx.Unlock()

			if l != 0 {
				return errors.New("a.conns is not empty")
			}

			return nil
		}))

		require.NoError(t, testWithTimeout(t, 5*time.Second, func() error {
			b.mx.Lock()
			l := len(b.conns)
			b.mx.Unlock()

			if l != 0 {
				return errors.New("b.conns is not empty")
			}

			return nil
		}))
	})

	t.Run("test transport establishment concurrently", func(t *testing.T) {
		initiatorsCount := 2
		remotesCount := 1

		initiators := make([]*Client, 0, initiatorsCount)
		remotes := make([]*Client, 0, remotesCount)

		// create initiators
		for i := 0; i < initiatorsCount; i++ {
			pk, sk := cipher.GenerateKeyPair()

			c := NewClient(pk, sk, dc)
			c.SetLogger(logging.MustGetLogger(fmt.Sprintf("Initiator %d", i)))
			err := c.InitiateServerConnections(context.Background(), 1)
			require.NoError(t, err)

			initiators = append(initiators, c)
		}

		// create remotes
		for i := 0; i < remotesCount; i++ {
			pk, sk := cipher.GenerateKeyPair()

			c := NewClient(pk, sk, dc)
			c.SetLogger(logging.MustGetLogger(fmt.Sprintf("Remote %d", i)))
			err := c.InitiateServerConnections(context.Background(), 1)
			require.NoError(t, err)

			remotes = append(remotes, c)
		}

		rand := rand.New(rand.NewSource(time.Now().UnixNano()))

		// store the number of transports each remote should handle
		usedRemotes := make(map[int]int)
		// mapping initiators to remotes
		pickedRemotes := make([]int, 0, initiatorsCount)
		for range initiators {
			remote := rand.Intn(remotesCount)
			if _, ok := usedRemotes[remote]; !ok {
				usedRemotes[remote] = 0
			}

			usedRemotes[remote] = usedRemotes[remote] + 1
			pickedRemotes = append(pickedRemotes, remote)
		}

		totalRemoteTpsCount := 0
		for _, connectionsCount := range usedRemotes {
			totalRemoteTpsCount += connectionsCount
		}

		acceptErrs := make(chan error, totalRemoteTpsCount)
		remotesTps := make(map[int][]transport.Transport, len(usedRemotes))
		var remotesWG sync.WaitGroup
		remotesWG.Add(totalRemoteTpsCount)
		for i, r := range remotes {
			if _, ok := usedRemotes[i]; ok {
				for connect := 0; connect < usedRemotes[i]; connect++ {
					// run remotes
					go func(remoteInd int) {
						var (
							transport transport.Transport
							err       error
						)

						transport, err = r.Accept(context.Background())
						if err != nil {
							acceptErrs <- err
						}

						remotesTps[remoteInd] = append(remotesTps[remoteInd], transport)

						remotesWG.Done()
					}(i)
				}
			}
		}

		dialErrs := make(chan error, initiatorsCount)
		initiatorsTps := make([]transport.Transport, 0, initiatorsCount)
		var initiatorsWG sync.WaitGroup
		initiatorsWG.Add(initiatorsCount)
		for i := range initiators {
			// run initiators
			go func(initiatorInd int) {
				var (
					transport transport.Transport
					err       error
				)

				transport, err = initiators[initiatorInd].Dial(context.Background(),
					remotes[pickedRemotes[initiatorInd]].pk)
				if err != nil {
					dialErrs <- err
				}

				initiatorsTps = append(initiatorsTps, transport)

				initiatorsWG.Done()
			}(i)
		}

		// wait for initiators
		initiatorsWG.Wait()
		close(dialErrs)
		err = <-dialErrs
		// single error should fail test
		require.NoError(t, err)

		// wait for remotes
		remotesWG.Wait()
		close(acceptErrs)
		err = <-acceptErrs
		// single error should fail test
		require.NoError(t, err)

		// check ServerConn's count
		require.Equal(t, len(usedRemotes)+initiatorsCount, len(s.conns))

		for i, initiator := range initiators {
			// get and check initiator's ServerConn
			initiatorServConn, ok := s.conns[initiator.pk]
			require.Equal(t, true, ok)
			require.Equal(t, initiator.pk, initiatorServConn.remoteClient)

			// get and check initiator's ClientConn
			initiatorClientConn, ok := initiator.conns[sPK]
			require.Equal(t, true, ok)
			require.Equal(t, sPK, initiatorClientConn.remoteSrv)

			remote := remotes[pickedRemotes[i]]

			// get and check remote's ServerConn
			remoteServConn, ok := s.conns[remote.pk]
			require.Equal(t, true, ok)
			require.Equal(t, remote.pk, remoteServConn.remoteClient)

			// get and check remote's ClientConn
			remoteClientConn, ok := remote.conns[sPK]
			require.Equal(t, true, ok)
			require.Equal(t, sPK, remoteClientConn.remoteSrv)

			// get initiator's nextConn
			initiatorNextConn := initiatorServConn.nextConns[initiatorClientConn.nextInitID-2]
			require.NotNil(t, initiatorNextConn)

			correspondingNextConnFound := false
			for nextConnID := remoteServConn.nextRespID - 2; nextConnID != remoteServConn.nextRespID; nextConnID -= 2 {
				if initiatorNextConn.id == nextConnID {
					correspondingNextConnFound = true
					break
				}
			}
			require.Equal(t, true, correspondingNextConnFound)

			correspondingNextConnFound = false
			for nextConnID := remoteServConn.nextRespID - 2; nextConnID != remoteServConn.nextRespID; nextConnID -= 2 {
				if remoteServConn.nextConns[nextConnID] != nil {
					if remoteServConn.nextConns[nextConnID].id == initiatorClientConn.nextInitID-2 {
						correspondingNextConnFound = true
						break
					}
				}
			}
			require.Equal(t, true, correspondingNextConnFound)
		}

		for _, tps := range remotesTps {
			for _, tp := range tps {
				err := tp.Close()
				require.NoError(t, err)
			}
		}

		for _, tp := range initiatorsTps {
			err := tp.Close()
			require.NoError(t, err)
		}

		for _, remote := range remotes {
			err := remote.Close()
			require.NoError(t, err)
		}

		for _, initiator := range initiators {
			err := initiator.Close()
			require.NoError(t, err)
		}

		require.NoError(t, testWithTimeout(t, 10*time.Second, func() error {
			s.mx.Lock()
			l := len(s.conns)
			s.mx.Unlock()

			if l != 0 {
				return errors.New("s.conns is not empty")
			}

			return nil
		}))

		for i, remote := range remotes {
			require.NoError(t, testWithTimeout(t, 10*time.Second, func() error {
				remote.mx.Lock()
				l := len(remote.conns)
				remote.mx.Unlock()

				if l != 0 {
					return errors.New(fmt.Sprintf("remotes[%v].conns is not empty", i))
				}

				return nil
			}))
		}

		for i, initiator := range initiators {
			require.NoError(t, testWithTimeout(t, 10*time.Second, func() error {
				initiator.mx.Lock()
				l := len(initiator.conns)
				initiator.mx.Unlock()

				if l != 0 {
					return errors.New(fmt.Sprintf("initiators[%v].conns is not empty", i))
				}

				return nil
			}))
		}
	})
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

	l, err := net.Listen("tcp", sAddr)
	require.NoError(t, err)

	log.Println(l.Addr().String())

	s, err := NewServer(sPK, sSK, l, dc)
	require.NoError(t, err)

	go s.Serve() //nolint:errcheck

	a := NewClient(aPK, aSK, dc)
	a.SetLogger(logging.MustGetLogger("A"))
	require.NoError(t, a.InitiateServerConnections(context.Background(), 1))

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
			require.NoError(t, err)
			require.Equal(t, len(pay), n)

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
	//wg.Wait()

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

func testWithTimeout(t *testing.T, timeout time.Duration, run func() error) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		if err := run(); err != nil {
			select {
			case <-timer.C:
				return err
			default:
				continue
			}
		}

		return nil
	}
}
