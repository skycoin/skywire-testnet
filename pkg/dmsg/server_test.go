package dmsg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
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

	go s.Serve() //nolint:errcheck

	// connect two clients, establish transport, check if there are
	// two ServerConn's and that both conn's `nextConn` is filled correctly
	t.Run("test transport establishment", func(t *testing.T) {
		aPK, aSK := cipher.GenerateKeyPair()
		bPK, bSK := cipher.GenerateKeyPair()

		a := NewClient(aPK, aSK, dc, SetLogger(logging.MustGetLogger("A")))
		err := a.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		b := NewClient(bPK, bSK, dc, SetLogger(logging.MustGetLogger("B")))
		err = b.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		bTransport, err := b.Dial(context.Background(), aPK)
		require.NoError(t, err)

		aTransport, err := a.Accept(context.Background())
		require.NoError(t, err)

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
		remotesTpCount := make(map[int]int)
		// mapping initiators to remotes. one initiator performs a single connection,
		// while remotes may handle from 0 to `initiatorsCount` connections
		pickedRemotes := make([]int, 0, initiatorsCount)
		for i := 0; i < initiatorsCount; i++ {
			// pick random remote, which the initiator will connect to
			remote := rand.Intn(remotesCount)
			// increment the number of connections picked remote will handle
			remotesTpCount[remote] = remotesTpCount[remote] + 1
			// map initiator to picked remote
			pickedRemotes = append(pickedRemotes, remote)
		}

		initiators := make([]*Client, 0, initiatorsCount)
		remotes := make([]*Client, 0, remotesCount)

		// create initiators
		for i := 0; i < initiatorsCount; i++ {
			pk, sk := cipher.GenerateKeyPair()

			c := NewClient(pk, sk, dc, SetLogger(logging.MustGetLogger(fmt.Sprintf("initiator_%d", i))))
			err := c.InitiateServerConnections(context.Background(), 1)
			require.NoError(t, err)

			initiators = append(initiators, c)
		}

		// create remotes
		for i := 0; i < remotesCount; i++ {
			pk, sk := cipher.GenerateKeyPair()

			c := NewClient(pk, sk, dc, SetLogger(logging.MustGetLogger(fmt.Sprintf("remote_%d", i))))
			if _, ok := remotesTpCount[i]; ok {
				err := c.InitiateServerConnections(context.Background(), 1)
				require.NoError(t, err)
			}
			remotes = append(remotes, c)
		}

		totalRemoteTpsCount := 0
		for _, connectionsCount := range remotesTpCount {
			totalRemoteTpsCount += connectionsCount
		}

		// channel to listen for `Accept` errors. Any single error must
		// fail the test
		acceptErrs := make(chan error, totalRemoteTpsCount)
		var remotesTpsMX sync.Mutex
		remotesTps := make(map[int][]transport.Transport, len(remotesTpCount))
		var remotesWG sync.WaitGroup
		remotesWG.Add(totalRemoteTpsCount)
		for i := range remotes {
			// only run `Accept` in case the remote was picked before
			if _, ok := remotesTpCount[i]; ok {
				for connect := 0; connect < remotesTpCount[i]; connect++ {
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
						remotesTpsMX.Lock()
						remotesTps[remoteInd] = append(remotesTps[remoteInd], transport)
						remotesTpsMX.Unlock()

						remotesWG.Done()
					}(i)
				}
			}
		}

		// channel to listen for `Dial` errors. Any single error must
		// fail the test
		dialErrs := make(chan error, initiatorsCount)
		var initiatorsTpsMx sync.Mutex
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
				initiatorsTpsMx.Lock()
				initiatorsTps = append(initiatorsTps, transport)
				initiatorsTpsMx.Unlock()

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
		require.Equal(t, len(remotesTpCount)+initiatorsCount, s.connCount())

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

	t.Run("failed_accepts_should_not_result_in_hang", func(t *testing.T) {
		// generate keys for both clients
		aPK, aSK := cipher.GenerateKeyPair()
		bPK, bSK := cipher.GenerateKeyPair()

		// create remote
		a := NewClient(aPK, aSK, dc, SetLogger(logging.MustGetLogger("A")))
		err = a.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		// create initiator
		b := NewClient(bPK, bSK, dc, SetLogger(logging.MustGetLogger("B")))
		err = b.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		bTransport, err := b.Dial(context.Background(), aPK)
		require.NoError(t, err)

		aTransport, err := a.Accept(context.Background())
		require.NoError(t, err)

		readWriteStop := make(chan struct{})
		readWriteDone := make(chan struct{})

		var readErr, writeErr error
		go func() {
			// read/write to/from transport until the stop signal arrives
			for {
				select {
				case <-readWriteStop:
					close(readWriteDone)
					return
				default:
					msg := []byte("Hello there!")
					if _, writeErr = bTransport.Write(msg); writeErr != nil {
						close(readWriteDone)
						return
					}
					if _, readErr = aTransport.Read(msg); readErr != nil {
						close(readWriteDone)
						return
					}
				}
			}
		}()

		// continue creating transports until the error occurs
		for {
			ctx := context.Background()
			if _, err = a.Dial(ctx, bPK); err != nil {
				break
			}
		}
		// must be error
		require.Error(t, err)

		// the same as above, transport is created by another client
		for {
			ctx := context.Background()
			if _, err = b.Dial(ctx, aPK); err != nil {
				break
			}
		}
		// must be error
		require.Error(t, err)

		// wait more time to ensure that the initially created transport works
		time.Sleep(2 * time.Second)

		err = aTransport.Close()
		require.NoError(t, err)

		err = bTransport.Close()
		require.NoError(t, err)

		// stop reading/writing goroutines
		close(readWriteStop)
		<-readWriteDone

		// check that the initial transport had been working properly all the time
		// if any error, it must be `io.EOF` for reader
		if readErr != io.EOF {
			require.NoError(t, readErr)
		}
		// if any error, it must be `io.ErrClosedPipe` for writer
		if writeErr != io.ErrClosedPipe {
			require.NoError(t, writeErr)
		}

		err = a.Close()
		require.NoError(t, err)

		err = b.Close()
		require.NoError(t, err)
	})

	t.Run("test sent/received message consistency", func(t *testing.T) {
		// generate keys for both clients
		aPK, aSK := cipher.GenerateKeyPair()
		bPK, bSK := cipher.GenerateKeyPair()

		// create remote
		a := NewClient(aPK, aSK, dc, SetLogger(logging.MustGetLogger("A")))
		err = a.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		// create initiator
		b := NewClient(bPK, bSK, dc, SetLogger(logging.MustGetLogger("B")))
		err = b.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		// create transports
		bTransport, err := b.Dial(context.Background(), aPK)
		require.NoError(t, err)

		aTransport, err := a.Accept(context.Background())
		require.NoError(t, err)

		msgCount := 100
		for i := 0; i < msgCount; i++ {
			msg := "Hello there!"

			// write message of 12 bytes
			_, err := bTransport.Write([]byte(msg))
			require.NoError(t, err)

			// create a receiving buffer of 5 bytes
			recBuff := make([]byte, 5)

			// read 5 bytes, 7 left
			n, err := aTransport.Read(recBuff)
			require.NoError(t, err)
			require.Equal(t, n, len(recBuff))

			received := string(recBuff[:n])

			// read 5 more, 2 left
			n, err = aTransport.Read(recBuff)
			require.NoError(t, err)
			require.Equal(t, n, len(recBuff))

			received += string(recBuff[:n])

			// read 2 bytes left
			n, err = aTransport.Read(recBuff)
			require.NoError(t, err)
			require.Equal(t, n, len(msg)-len(recBuff)*2)

			received += string(recBuff[:n])

			// received string must be equal to the sent one
			require.Equal(t, received, msg)
		}

		err = bTransport.Close()
		require.NoError(t, err)

		err = aTransport.Close()
		require.NoError(t, err)

		err = a.Close()
		require.NoError(t, err)

		err = b.Close()
		require.NoError(t, err)
	})

	t.Run("capped_transport_buffer_should_not_result_in_hang", func(t *testing.T) {
		// generate keys for both clients
		aPK, aSK := cipher.GenerateKeyPair()
		bPK, bSK := cipher.GenerateKeyPair()

		// create remote
		a := NewClient(aPK, aSK, dc, SetLogger(logging.MustGetLogger("A")))
		err = a.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		// create initiator
		b := NewClient(bPK, bSK, dc, SetLogger(logging.MustGetLogger("B")))
		err = b.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		// create transports
		aWrTransport, err := a.Dial(context.Background(), bPK)
		require.NoError(t, err)

		_, err = b.Accept(context.Background())
		require.NoError(t, err)

		msg := []byte("Hello there!")
		// exact iterations to fill the receiving buffer and hang `Write`
		iterationsToDo := tpBufCap/len(msg) + 1

		// fill the buffer, but no block yet
		for i := 0; i < iterationsToDo-1; i++ {
			_, err = aWrTransport.Write(msg)
			require.NoError(t, err)
		}

		// block on `Write`
		go func() {
			_, err = aWrTransport.Write(msg)
			require.Error(t, err)
		}()

		// wait till it's definitely blocked
		time.Sleep(1 * time.Second)

		// create new transport from `B` to `A`
		bWrTransport, err := b.Dial(context.Background(), aPK)
		require.NoError(t, err)

		aRdTransport, err := a.Accept(context.Background())
		require.NoError(t, err)

		// try to write/read message via the new transports
		for i := 0; i < 100; i++ {
			_, err := bWrTransport.Write(msg)
			require.NoError(t, err)

			recBuff := make([]byte, len(msg))
			_, err = aRdTransport.Read(recBuff)
			require.NoError(t, err)

			require.Equal(t, recBuff, msg)
		}

		err = aWrTransport.Close()
		require.NoError(t, err)

		err = bWrTransport.Close()
		require.NoError(t, err)

		err = aRdTransport.Close()
		require.NoError(t, err)

		err = a.Close()
		require.NoError(t, err)

		err = b.Close()
		require.NoError(t, err)
	})

	t.Run("self_dial_should_work", func(t *testing.T) {
		// generate keys for the client
		aPK, aSK := cipher.GenerateKeyPair()

		// create client
		a := NewClient(aPK, aSK, dc, SetLogger(logging.MustGetLogger("A")))
		err = a.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		// self-dial
		selfWrTp, err := a.Dial(context.Background(), aPK)
		require.NoError(t, err)

		// self-accept
		selfRdTp, err := a.Accept(context.Background())
		require.NoError(t, err)

		// try to write/read message to/from self
		msgCount := 100
		for i := 0; i < msgCount; i++ {
			msg := []byte("Hello there!")

			_, err := selfWrTp.Write(msg)
			require.NoError(t, err)

			recBuf := make([]byte, 5)

			_, err = selfRdTp.Read(recBuf)
			require.NoError(t, err)

			_, err = selfRdTp.Read(recBuf)
			require.NoError(t, err)

			_, err = selfRdTp.Read(recBuf)
			require.NoError(t, err)
		}

		err = selfRdTp.Close()
		require.NoError(t, err)

		err = selfWrTp.Close()
		require.NoError(t, err)

		err = a.Close()
		require.NoError(t, err)
	})

	t.Run("server_disconnect_should_close_transports", func(t *testing.T) {
		// generate keys for server
		sPK, sSK := cipher.GenerateKeyPair()

		dc := client.NewMock()

		l, err := nettest.NewLocalListener("tcp")
		require.NoError(t, err)

		// create a server separately from other tests, since this one should be closed
		s, err := NewServer(sPK, sSK, "", l, dc)
		require.NoError(t, err)

		var sStartErr error
		sDone := make(chan struct{})
		go func() {
			if err := s.Serve(); err != nil {
				sStartErr = err
			}

			sDone <- struct{}{}
		}()

		// generate keys for both clients
		aPK, aSK := cipher.GenerateKeyPair()
		bPK, bSK := cipher.GenerateKeyPair()

		// create remote
		a := NewClient(aPK, aSK, dc, SetLogger(logging.MustGetLogger("A")))
		err = a.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		// create initiator
		b := NewClient(bPK, bSK, dc, SetLogger(logging.MustGetLogger("B")))
		err = b.InitiateServerConnections(context.Background(), 1)
		require.NoError(t, err)

		bTransport, err := b.Dial(context.Background(), aPK)
		require.NoError(t, err)

		aTransport, err := a.Accept(context.Background())
		require.NoError(t, err)

		msgCount := 100
		for i := 0; i < msgCount; i++ {
			msg := []byte("Hello there!")

			_, err := bTransport.Write(msg)
			require.NoError(t, err)

			recBuff := make([]byte, 5)

			_, err = aTransport.Read(recBuff)
			require.NoError(t, err)

			_, err = aTransport.Read(recBuff)
			require.NoError(t, err)

			_, err = aTransport.Read(msg)
			require.NoError(t, err)
		}

		err = s.Close()
		require.NoError(t, err)

		<-sDone
		// TODO: remove log, uncomment when bug is fixed
		log.Printf("SERVE ERR: %v", sStartErr)
		//require.NoError(t, sStartErr)

		/*time.Sleep(10 * time.Second)

		tp, ok := bTransport.(*Transport)
		require.Equal(t, true, ok)
		require.Equal(t, true, tp.IsClosed())

		tp, ok = aTransport.(*Transport)
		require.Equal(t, true, ok)
		require.Equal(t, true, tp.IsClosed())*/
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

	remote := NewClient(remotePK, remoteSK, dc, SetLogger(logging.MustGetLogger("remote")))
	err = remote.InitiateServerConnections(ctx, 1)
	require.NoError(t, err)

	require.NoError(t, testWithTimeout(smallDelay, func() error {
		if s.connCount() != 1 {
			return errors.New("s.conns is not equal to 1")
		}
		return nil
	}))

	initiator := NewClient(initiatorPK, initiatorSK, dc, SetLogger(logging.MustGetLogger("initiator")))
	err = initiator.InitiateServerConnections(ctx, 1)
	require.NoError(t, err)

	initiatorTransport, err := initiator.Dial(ctx, remotePK)
	require.NoError(t, err)

	remoteTransport, err := remote.Accept(context.Background())
	require.NoError(t, err)

	require.NoError(t, testWithTimeout(smallDelay, func() error {
		if s.connCount() != 2 {
			return errors.New("s.conns is not equal to 2")
		}
		return nil
	}))

	err = s.Close()
	assert.NoError(t, err)

	initTr := initiatorTransport.(*Transport)
	assert.False(t, isDoneChannelOpen(initTr.done))
	assert.False(t, isReadChannelOpen(initTr.inCh))

	remoteTr := remoteTransport.(*Transport)
	assert.False(t, isDoneChannelOpen(remoteTr.done))
	assert.False(t, isReadChannelOpen(remoteTr.inCh))

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

	require.NoError(t, testWithTimeout(smallDelay, func() error {
		_, err = initiator.Dial(ctx, remotePK)
		if err != nil {
			return err
		}

		_, err = remote.Accept(context.Background())
		return err
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
	sAddr := "127.0.0.1:8081"

	const tpCount = 10
	const msgCount = 100

	dc := client.NewMock()

	l, err := net.Listen("tcp", sAddr)
	require.NoError(t, err)

	log.Println(l.Addr().String())

	s, err := NewServer(sPK, sSK, "", l, dc)
	require.NoError(t, err)

	go s.Serve() //nolint:errcheck

	a := NewClient(aPK, aSK, dc, SetLogger(logging.MustGetLogger("A")))
	require.NoError(t, a.InitiateServerConnections(context.Background(), 1))

	b := NewClient(bPK, bSK, dc, SetLogger(logging.MustGetLogger("B")))
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
