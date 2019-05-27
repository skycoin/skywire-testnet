package messaging

import (
	"context"
	"errors"
	"net"
	"os"
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

func TestMain(m *testing.M) {
	lvl, _ := logging.LevelFromString("error") // nolint: errcheck
	logging.SetLevel(lvl)
	os.Exit(m.Run())
}

func TestClientDial(t *testing.T) {
	discovery := client.NewMock()

	srv, err := newMockServer(discovery)
	require.NoError(t, err)
	srvPK := srv.config.Public

	pk1, sk1 := cipher.GenerateKeyPair()
	c1 := NewClient(&Config{pk1, sk1, discovery, 0, 0})

	pk2, sk2 := cipher.GenerateKeyPair()
	c2 := NewClient(&Config{pk2, sk2, discovery, 0, 0})
	require.NoError(t, c2.ConnectToInitialServers(context.TODO(), 1))

	var (
		tp2     transport.Transport
		tp2Err  error
		tp2Done = make(chan struct{})
	)
	go func() {
		tp2, tp2Err = c2.Accept(context.TODO())
		close(tp2Done)
	}()

	var (
		tp1     transport.Transport
		tp1Err  error
		tp1Done = make(chan struct{})
	)
	go func() {
		tp1, tp1Err = c1.Dial(context.TODO(), pk2)
		close(tp1Done)
	}()

	<-tp1Done
	require.NoError(t, tp1Err)
	require.NotNil(t, c1.getLink(srvPK).chans.get(0))

	entry, err := discovery.Entry(context.TODO(), pk1)
	require.NoError(t, err)
	require.Len(t, entry.Client.DelegatedServers, 1)

	<-tp2Done
	require.NoError(t, tp2Err)
	require.NotNil(t, c2.getLink(srvPK).chans.get(0))

	go tp1.Write([]byte("foo")) // nolint: errcheck

	buf := make([]byte, 3)
	n, err := tp2.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)

	go tp2.Write([]byte("bar")) // nolint: errcheck

	buf = make([]byte, 3)
	n, err = tp1.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("bar"), buf)

	require.NoError(t, tp1.Close())
	require.NoError(t, tp2.Close())

	// It is expected for the messaging client to delete the channel for chanList eventually.
	require.True(t, retry(time.Second*10, time.Second, func() bool {
		return c2.getLink(srvPK).chans.get(0) == nil
	}))
}

// retries until successful under a given deadline.
// 'tick' specifies the break duration before retry.
func retry(deadline, tick time.Duration, do func() bool) bool {
	timer := time.NewTimer(deadline)
	defer timer.Stop()

	done := make(chan struct{})
	doneOnce := new(sync.Once)
	defer doneOnce.Do(func() { close(done) })

	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.Tick(tick):
				if ok := do(); ok {
					doneOnce.Do(func() { close(done) })
					return
				}
			}
		}
	}()

	for {
		select {
		case <-timer.C:
			return false
		case <-done:
			return true
		}
	}
}

type mockServer struct {
	*Pool

	links []*Link
	mu    sync.Mutex
}

func newMockServer(discovery client.APIClient) (*mockServer, error) {
	pk, sk := cipher.GenerateKeyPair()
	srv := &mockServer{}
	pool := NewPool(DefaultLinkConfig(), &Callbacks{
		HandshakeComplete: srv.onHandshake,
		Data:              srv.onData,
	})
	pool.config.Public = pk
	pool.config.Secret = sk

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	go pool.Respond(l) // nolint: errcheck

	entry := client.NewServerEntry(pk, 0, l.Addr().String(), 10)
	if err := entry.Sign(sk); err != nil {
		return nil, err
	}
	if err := discovery.SetEntry(context.TODO(), entry); err != nil {
		return nil, err
	}

	srv.Pool = pool
	return srv, nil
}

func (s *mockServer) onHandshake(l *Link) {
	s.mu.Lock()
	s.links = append(s.links, l)
	s.mu.Unlock()
}

func (s *mockServer) onData(l *Link, frameType FrameType, body []byte) error {
	ol, err := s.oppositeLink(l)
	if err != nil {
		return err
	}

	channelID := body[0]
	switch frameType {
	case FrameTypeOpenChannel:
		_, err = ol.SendOpenChannel(channelID, l.Remote(), body[34:])
	case FrameTypeChannelOpened:
		_, err = ol.SendChannelOpened(channelID, channelID, body[2:])
	case FrameTypeCloseChannel:
		l.SendChannelClosed(channelID) // nolint
		_, err = ol.SendCloseChannel(channelID)
	case FrameTypeChannelClosed:
		_, err = ol.SendChannelClosed(channelID)
	case FrameTypeSend:
		_, err = ol.Send(channelID, body[1:])
	}

	return err
}

func (s *mockServer) oppositeLink(l *Link) (*Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, link := range s.links {
		if link != l {
			return link, nil
		}
	}

	return nil, errors.New("unknown link")
}
