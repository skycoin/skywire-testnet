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

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/transport"
)

// TestServerConn_AddNext ensures that `nextConns` for the remote client is being filled correctly.
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

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	cases := []struct {
		name string
		conn *ServerConn
		ctx  context.Context
		want want
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
		_, ok := serverConn.getNext(i)
		require.Equal(t, true, ok)
	}

	for i := uint16(connsCount*2 + 1); i != 1; i += 2 {
		_, ok := serverConn.getNext(i)
		require.Equal(t, false, ok)
	}
}

// TestNewServer ensures Server starts and quits with no error.
func TestNewServer(t *testing.T) {
	sPK, sSK := cipher.GenerateKeyPair()
	dc := client.NewMock()

	l, err := net.Listen("tcp", "")
	require.NoError(t, err)

	// When calling 'NewServer', if the provided net.Listener is already a noise.Listener,
	// An error should be returned.
	t.Run("fail_on_wrapped_listener", func(t *testing.T) {
		wrappedL := noise.WrapListener(l, sPK, sSK, false, noise.HandshakeXK)
		s, err := NewServer(sPK, sSK, "", wrappedL, dc)
		assert.Equal(t, ErrListenerAlreadyWrappedToNoise, err)
		assert.Nil(t, s)
	})

	s, err := NewServer(sPK, sSK, "", l, dc)
	require.NoError(t, err)

	go s.Serve() //nolint:errcheck

	time.Sleep(time.Second)

	assert.NoError(t, s.Close())
}

// TestServer_Serve ensures that Server processes request frames and
// instantiates transports properly.
func TestServer_Serve(t *testing.T) {
	sPK, sSK := cipher.GenerateKeyPair()
	dc := client.NewMock()

	l, err := nettest.NewLocalListener("tcp")
	require.NoError(t, err)

	s, err := NewServer(sPK, sSK, "", l, dc)
	require.NoError(t, err)

	go s.Serve() // nolint:errcheck

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
		require.Equal(t, 2, s.connCount())

		// must have ServerConn for A
		aServerConn, ok := s.getConn(aPK)
		require.Equal(t, true, ok)
		require.Equal(t, aPK, aServerConn.PK())

		// must have ServerConn for B
		bServerConn, ok := s.getConn(bPK)
		require.Equal(t, true, ok)
		require.Equal(t, bPK, bServerConn.PK())

		// must have a ClientConn
		aClientConn, ok := a.getConn(sPK)
		require.Equal(t, true, ok)
		require.Equal(t, sPK, aClientConn.RemotePK())

		// must have a ClientConn
		bClientConn, ok := b.getConn(sPK)
		require.Equal(t, true, ok)
		require.Equal(t, sPK, bClientConn.RemotePK())

		// check whether nextConn's contents are as must be
		bClientConn.mx.RLock()
		nextInitID := bClientConn.nextInitID
		bClientConn.mx.RUnlock()
		bNextConn, ok := bServerConn.getNext(nextInitID - 2)
		require.Equal(t, true, ok)
		aServerConn.mx.RLock()
		nextRespID := aServerConn.nextRespID
		aServerConn.mx.RUnlock()
		require.Equal(t, bNextConn.id, nextRespID-2)

		// check whether nextConn's contents are as must be
		aServerConn.mx.RLock()
		nextRespID = aServerConn.nextRespID
		aServerConn.mx.RUnlock()
		aNextConn, ok := aServerConn.getNext(nextRespID - 2)
		require.Equal(t, true, ok)
		bClientConn.mx.RLock()
		nextInitID = bClientConn.nextInitID
		bClientConn.mx.RUnlock()
		require.Equal(t, aNextConn.id, nextInitID-2)

		err = aTransport.Close()
		require.NoError(t, err)

		err = bTransport.Close()
		require.NoError(t, err)

		err = a.Close()
		require.NoError(t, err)

		err = b.Close()
		require.NoError(t, err)

		require.NoError(t, testWithTimeout(5*time.Second, func() error {
			if s.connCount() != 0 {
				return errors.New("s.conns is not empty")
			}

			return nil
		}))

		require.NoError(t, testWithTimeout(5*time.Second, func() error {
			if a.connCount() != 0 {
				return errors.New("a.conns is not empty")
			}

			return nil
		}))

		require.NoError(t, testWithTimeout(5*time.Second, func() error {
			if b.connCount() != 0 {
				return errors.New("b.conns is not empty")
			}

			return nil
		}))
	})

	t.Run("test transport establishment concurrently", func(t *testing.T) {
		// this way we can control the tests' difficulty
		initiatorsCount := 50
		remotesCount := 50

		rand := rand.New(rand.NewSource(time.Now().UnixNano()))

		// store the number of transports each remote should handle
		usedRemotes := make(map[int]int)
		// mapping initiators to remotes. one initiator performs a single connection,
		// while remotes may handle from 0 to `initiatorsCount` connections
		pickedRemotes := make([]int, 0, initiatorsCount)
		for i := 0; i < initiatorsCount; i++ {
			// pick random remote, which the initiator will connect to
			remote := rand.Intn(remotesCount)
			// increment the number of connections picked remote will handle
			usedRemotes[remote] = usedRemotes[remote] + 1
			// map initiator to picked remote
			pickedRemotes = append(pickedRemotes, remote)
		}

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
			if _, ok := usedRemotes[i]; ok {
				err := c.InitiateServerConnections(context.Background(), 1)
				require.NoError(t, err)
			}
			remotes = append(remotes, c)
		}

		totalRemoteTpsCount := 0
		for _, connectionsCount := range usedRemotes {
			totalRemoteTpsCount += connectionsCount
		}

		// channel to listen for `Accept` errors. Any single error must
		// fail the test
		acceptErrs := make(chan error, totalRemoteTpsCount)
		remotesTps := make(map[int][]transport.Transport, len(usedRemotes))
		var remotesWG sync.WaitGroup
		remotesWG.Add(totalRemoteTpsCount)
		for i := range remotes {
			// only run `Accept` in case the remote was picked before
			if _, ok := usedRemotes[i]; ok {
				for connect := 0; connect < usedRemotes[i]; connect++ {
					// run remote
					go func(remoteInd int) {
						var (
							transport transport.Transport
							err       error
						)

						transport, err = remotes[remoteInd].Accept(context.Background())
						if err != nil {
							acceptErrs <- err
						}

						// store transport
						remotesTps[remoteInd] = append(remotesTps[remoteInd], transport)

						remotesWG.Done()
					}(i)
				}
			}
		}

		// channel to listen for `Dial` errors. Any single error must
		// fail the test
		dialErrs := make(chan error, initiatorsCount)
		initiatorsTps := make([]transport.Transport, 0, initiatorsCount)
		var initiatorsWG sync.WaitGroup
		initiatorsWG.Add(initiatorsCount)
		for i := range initiators {
			// run initiator
			go func(initiatorInd int) {
				var (
					transport transport.Transport
					err       error
				)

				remote := remotes[pickedRemotes[initiatorInd]]
				transport, err = initiators[initiatorInd].Dial(context.Background(), remote.pk)
				if err != nil {
					dialErrs <- err
				}

				// store transport
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
		require.Equal(t, len(usedRemotes)+initiatorsCount, s.connCount())

		for i, initiator := range initiators {
			// get and check initiator's ServerConn
			initiatorServConn, ok := s.getConn(initiator.pk)
			require.Equal(t, true, ok)
			require.Equal(t, initiator.pk, initiatorServConn.PK())

			// get and check initiator's ClientConn
			initiatorClientConn, ok := initiator.getConn(sPK)
			require.Equal(t, true, ok)
			require.Equal(t, sPK, initiatorClientConn.RemotePK())

			remote := remotes[pickedRemotes[i]]

			// get and check remote's ServerConn
			remoteServConn, ok := s.getConn(remote.pk)
			require.Equal(t, true, ok)
			require.Equal(t, remote.pk, remoteServConn.PK())

			// get and check remote's ClientConn
			remoteClientConn, ok := remote.getConn(sPK)
			require.Equal(t, true, ok)
			require.Equal(t, sPK, remoteClientConn.RemotePK())

			// get initiator's nextConn
			initiatorClientConn.mx.RLock()
			nextInitID := initiatorClientConn.nextInitID
			initiatorClientConn.mx.RUnlock()
			initiatorNextConn, ok := initiatorServConn.getNext(nextInitID - 2)
			require.Equal(t, true, ok)
			require.NotNil(t, initiatorNextConn)

			// since, the test is concurrent, we can not check the exact values, so
			// we just loop through the previous values of `nextRespID` and check
			// whether one of these is equal to the corresponding value of `initiatorNextConn.id`
			correspondingNextConnFound := false
			remoteServConn.mx.RLock()
			nextConnID := remoteServConn.nextRespID
			remoteServConn.mx.RUnlock()
			for i := nextConnID - 2; i != nextConnID; i -= 2 {
				if initiatorNextConn.id == i {
					correspondingNextConnFound = true
					break
				}
			}
			require.Equal(t, true, correspondingNextConnFound)

			// same as above. Looping through the previous values of `nextRespID`,
			// fetching all of the corresponding `nextConn`. One of these must have `id`
			// equal to `initiatorClientConn.nextInitID - 2`
			correspondingNextConnFound = false
			remoteServConn.mx.RLock()
			nextConnID = remoteServConn.nextRespID
			remoteServConn.mx.RUnlock()
			for i := nextConnID - 2; i != nextConnID; i -= 2 {
				if _, ok := remoteServConn.getNext(i); ok {
					initiatorClientConn.mx.RLock()
					initiatorNextInitID := initiatorClientConn.nextInitID - 2
					initiatorClientConn.mx.RUnlock()
					if next, ok := remoteServConn.getNext(i); ok && next.id == initiatorNextInitID {
						correspondingNextConnFound = true
						break
					}
				}
			}
			require.Equal(t, true, correspondingNextConnFound)
		}

		// close transports for remotes
		for _, tps := range remotesTps {
			for _, tp := range tps {
				err := tp.Close()
				require.NoError(t, err)
			}
		}

		// close transports for initiators
		for _, tp := range initiatorsTps {
			err := tp.Close()
			require.NoError(t, err)
		}

		// close remotes
		for _, remote := range remotes {
			err := remote.Close()
			require.NoError(t, err)
		}

		// close initiators
		for _, initiator := range initiators {
			err := initiator.Close()
			require.NoError(t, err)
		}

		require.NoError(t, testWithTimeout(10*time.Second, func() error {
			if s.connCount() != 0 {
				return errors.New("s.conns is not empty")
			}

			return nil
		}))

		for i, remote := range remotes {
			require.NoError(t, testWithTimeout(10*time.Second, func() error {
				if remote.connCount() != 0 {
					return fmt.Errorf("remotes[%v].conns is not empty", i)
				}

				return nil
			}))
		}

		for i, initiator := range initiators {
			require.NoError(t, testWithTimeout(10*time.Second, func() error {
				if initiator.connCount() != 0 {
					return fmt.Errorf("initiators[%v].conns is not empty", i)
				}

				return nil
			}))
		}
	})

	t.Run("test failed accept not hanging already established transport", func(t *testing.T) {
		// generate keys for both clients
		aPK, aSK := cipher.GenerateKeyPair()
		bPK, bSK := cipher.GenerateKeyPair()

		// create remote
		a := NewClient(aPK, aSK, dc)
		a.SetLogger(logging.MustGetLogger("A"))
		err := a.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		// create initiator
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

		aTpDone := make(chan struct{})
		bTpDone := make(chan struct{})

		var bErr error
		var tpReadWriteWG sync.WaitGroup
		tpReadWriteWG.Add(2)
		// run infinite reading from tp loop in goroutine
		go func() {
			for {
				select {
				case <-aTpDone:
					log.Println("ATransport DONE")
					tpReadWriteWG.Done()
					return
				default:
					msg := make([]byte, 13)
					if _, aErr = aTransport.Read(msg); aErr != nil {
						tpReadWriteWG.Done()
						return
					}
					log.Printf("GOT MESSAGE %s", string(msg))
				}
			}
		}()

		// run infinite writing to tp loop in goroutine
		go func() {
			for {
				select {
				case <-bTpDone:
					log.Println("BTransport DONE")
					tpReadWriteWG.Done()
					return
				default:
					msg := []byte("Hello there!")
					if _, bErr = bTransport.Write(msg); bErr != nil {
						tpReadWriteWG.Done()
						return
					}
				}
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// try to create another transport
		_, err = a.Dial(ctx, bPK)
		// must fail with timeout
		require.Error(t, err)

		// wait more time to ensure that the initially created transport works
		time.Sleep(2 * time.Second)

		// stop reading/writing goroutines
		close(aTpDone)
		close(bTpDone)

		// wait for goroutines to stop
		tpReadWriteWG.Wait()
		// check that the initial transport had been working properly all the time
		require.NoError(t, aErr)
		require.NoError(t, bErr)

		err = aTransport.Close()
		require.NoError(t, err)

		err = bTransport.Close()
		require.NoError(t, err)

		err = a.Close()
		require.NoError(t, err)

		err = b.Close()
		require.NoError(t, err)
	})

	t.Run("Reconnect to server should succeed", func(t *testing.T) {
		t.Run("Same address", func(t *testing.T) {
			t.Parallel()
			testReconnect(t, false)
		})

		t.Run("Random address", func(t *testing.T) {
			t.Parallel()
			testReconnect(t, true)
		})
	})
}

func testReconnect(t *testing.T, randomAddr bool) {
	const smallDelay = 100 * time.Millisecond
	ctx := context.TODO()

	serverPK, serverSK := cipher.GenerateKeyPair()
	dc := client.NewMock()

	l, err := nettest.NewLocalListener("tcp")
	require.NoError(t, err)

	s, err := NewServer(serverPK, serverSK, "", l, dc)
	require.NoError(t, err)

	serverAddr := s.Addr()

	go s.Serve() // nolint:errcheck

	remotePK, remoteSK := cipher.GenerateKeyPair()
	initiatorPK, initiatorSK := cipher.GenerateKeyPair()

	assert.Equal(t, 0, s.connCount())

	remote := NewClient(remotePK, remoteSK, dc)
	remote.SetLogger(logging.MustGetLogger("remote"))
	err = remote.InitiateServerConnections(ctx, 1)
	require.NoError(t, err)

	require.NoError(t, testWithTimeout(smallDelay, func() error {
		if s.connCount() != 1 {
			return errors.New("s.conns is not equal to 1")
		}
		return nil
	}))

	initiator := NewClient(initiatorPK, initiatorSK, dc)
	initiator.SetLogger(logging.MustGetLogger("initiator"))
	err = initiator.InitiateServerConnections(ctx, 1)
	require.NoError(t, err)

	require.NoError(t, testWithTimeout(smallDelay, func() error {
		if s.connCount() != 2 {
			return errors.New("s.conns is not equal to 2")
		}
		return nil
	}))

	err = s.Close()
	assert.NoError(t, err)

	assert.Equal(t, 0, s.connCount())

	addr := ""
	if !randomAddr {
		addr = serverAddr
	}

	l, err = net.Listen("tcp", serverAddr)
	require.NoError(t, err)

	s, err = NewServer(serverPK, serverSK, addr, l, dc)
	require.NoError(t, err)

	go s.Serve() // nolint:errcheck

	require.NoError(t, testWithTimeout(clientReconnectInterval+smallDelay, func() error {
		if s.connCount() != 2 {
			return errors.New("s.conns is not equal to 2")
		}
		return nil
	}))

	err = s.Close()
	assert.NoError(t, err)
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

	s, err := NewServer(sPK, sSK, "", l, dc)
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
	wg.Wait()

	// Close server.
	assert.NoError(t, s.Close())
}

func catch(err error) {
	if err != nil {
		panic(err)
	}
}

// intended to test some func of `func() error` signature with a given timeout.
// Exeeding timeout results in error.
func testWithTimeout(timeout time.Duration, run func() error) error {
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