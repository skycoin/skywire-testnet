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

func TestClientConnectInitialServers(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	discovery := client.NewMock()
	c := NewClient(pk, sk, discovery)

	srv, err := newMockServer(discovery)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, c.ConnectToInitialServers(context.TODO(), 1))
	c.mu.RLock()
	require.Len(t, c.links, 1)
	c.mu.RUnlock()

	entry, err := discovery.Entry(context.TODO(), pk)
	require.NoError(t, err)
	assert.Len(t, entry.Client.DelegatedServers, 1)
	assert.Equal(t, srv.config.Public, entry.Client.DelegatedServers[0])

	require.NoError(t, srv.Close())
	time.Sleep(100 * time.Millisecond)

	c.mu.RLock()
	require.Len(t, c.links, 0)
	c.mu.RUnlock()

	entry, err = discovery.Entry(context.TODO(), pk)
	require.NoError(t, err)
	require.Len(t, entry.Client.DelegatedServers, 0)
}

func TestClientDial(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	discovery := client.NewMock()
	c := NewClient(pk, sk, discovery)

	srv, err := newMockServer(discovery)
	require.NoError(t, err)
	srvPK := srv.config.Public

	time.Sleep(100 * time.Millisecond)

	anotherPK, anotherSK := cipher.GenerateKeyPair()
	anotherClient := NewClient(anotherPK, anotherSK, discovery)
	require.NoError(t, anotherClient.ConnectToInitialServers(context.TODO(), 1))

	var anotherTr transport.Transport
	anotherErrCh := make(chan error)
	go func() {
		t, err := anotherClient.Accept(context.TODO())
		anotherTr = t
		anotherErrCh <- err
	}()

	var tr transport.Transport
	errCh := make(chan error)
	go func() {
		t, err := c.Dial(context.TODO(), anotherPK)
		tr = t
		errCh <- err
	}()

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, <-errCh)
	require.NotNil(t, c.getLink(srvPK).chans.get(0))

	entry, err := discovery.Entry(context.TODO(), pk)
	require.NoError(t, err)
	require.Len(t, entry.Client.DelegatedServers, 1)

	require.NoError(t, <-anotherErrCh)
	require.NotNil(t, anotherClient.getLink(srvPK).chans.get(0))

	go tr.Write([]byte("foo")) // nolint: errcheck

	buf := make([]byte, 3)
	n, err := anotherTr.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)

	go anotherTr.Write([]byte("bar")) // nolint: errcheck

	buf = make([]byte, 3)
	n, err = tr.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("bar"), buf)

	require.NoError(t, anotherTr.Close())
	require.NoError(t, tr.Close())

	time.Sleep(100 * time.Millisecond)

	// require.Nil(t, c.getLink(srvPK).chans.get(0))
	require.Nil(t, anotherClient.getLink(srvPK).chans.get(0))
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
		_, err = ol.SendChannelOpened(channelID, body[1:])
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
