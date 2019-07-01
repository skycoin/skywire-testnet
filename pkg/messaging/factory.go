// Package messaging implements messaging communication. Messaging
// communication is performed between 2 nodes using intermediate relay
// server, visor discovery is performed using messaging discovery.
package messaging

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/cipher"

	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/transport"
)

var (
	// ErrNoSrv indicate that remote client does not have DelegatedServers in entry.
	ErrNoSrv = errors.New("remote has no DelegatedServers")
	// ErrRejected indicates that ChannelOpen frame was rejected by remote server.
	ErrRejected = errors.New("rejected")
	// ErrChannelClosed indicates that underlying channel is being closed and writes are prohibited.
	ErrChannelClosed = errors.New("messaging channel closed")
	// ErrDeadlineExceeded indicates that read/write operation failed due to timeout.
	ErrDeadlineExceeded = errors.New("deadline exceeded in messaging")
	// ErrClientClosed indicates that client is closed and not accepting new connections.
	ErrClientClosed = errors.New("client closed")
)

type clientLink struct {
	link  *Link
	addr  string
	chans *chanList
}

// Config configures MsgFactory
type Config struct {
	PubKey     cipher.PubKey
	SecKey     cipher.SecKey
	Discovery  client.APIClient
	Retries    int
	RetryDelay time.Duration
}

// MsgFactory sends messages to remote client nodes via relay Server
// Implements Factory
type MsgFactory struct {
	Logger *logging.Logger

	pubKey cipher.PubKey
	secKey cipher.SecKey
	dc     client.APIClient
	pool   *Pool

	retries    int
	retryDelay time.Duration

	links map[cipher.PubKey]*clientLink
	mu    sync.RWMutex

	newCh chan *msgChannel // chan for newly opened channels
	newWG sync.WaitGroup   // waits for goroutines writing to newCh to end.

	doneCh chan struct{}
}

// NewMsgFactory constructs a new MsgFactory
func NewMsgFactory(conf *Config) *MsgFactory {
	msgFactory := &MsgFactory{
		Logger:     logging.MustGetLogger("messenger"),
		pubKey:     conf.PubKey,
		secKey:     conf.SecKey,
		dc:         conf.Discovery,
		retries:    conf.Retries,
		retryDelay: conf.RetryDelay,
		links:      make(map[cipher.PubKey]*clientLink),
		newCh:      make(chan *msgChannel),
		doneCh:     make(chan struct{}),
	}
	config := &LinkConfig{
		Public:           msgFactory.pubKey,
		Secret:           msgFactory.secKey,
		HandshakeTimeout: DefaultHandshakeTimeout,
	}
	msgFactory.pool = NewPool(config, &Callbacks{
		Data:  msgFactory.onData,
		Close: msgFactory.onClose,
	})

	return msgFactory
}

// ConnectToInitialServers tries to connect to at most serverCount servers.
func (msgFactory *MsgFactory) ConnectToInitialServers(ctx context.Context, serverCount int) error {
	if serverCount == 0 {
		return nil
	}

	entries, err := msgFactory.dc.AvailableServers(ctx)
	if err != nil {
		return fmt.Errorf("servers are not available: %s", err)
	}

	for _, entry := range entries {
		if len(msgFactory.links) > serverCount {
			break
		}

		if _, err := msgFactory.link(entry.Static, entry.Server.Address); err != nil {
			msgFactory.Logger.Warnf("Failed to connect to the server %s: %s", entry.Static, err)
		}
	}

	if len(msgFactory.links) == 0 {
		return fmt.Errorf("servers are not available: all servers failed")
	}

	if err := msgFactory.setEntry(ctx); err != nil {
		return fmt.Errorf("entry update failure: %s", err)
	}

	return nil
}

// Accept accepts a remotely-initiated Transport.
func (msgFactory *MsgFactory) Accept(ctx context.Context) (transport.Transport, error) {
	select {
	case ch, more := <-msgFactory.newCh:
		if !more {
			return nil, ErrClientClosed
		}
		return ch, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Dial initiates a Transport with a remote visor.
func (msgFactory *MsgFactory) Dial(ctx context.Context, remote cipher.PubKey) (transport.Transport, error) {
	entry, err := msgFactory.dc.Entry(ctx, remote)
	if err != nil {
		return nil, fmt.Errorf("get entry failure: %s", err)
	}

	if entry.Client.DelegatedServers == nil || len(entry.Client.DelegatedServers) == 0 {
		return nil, ErrNoSrv
	}

	clientLink, err := msgFactory.ensureLink(ctx, entry.Client.DelegatedServers)
	if err != nil {
		return nil, fmt.Errorf("link failure: %s", err)
	}

	channel, err := newChannel(true, msgFactory.secKey, remote, clientLink.link)
	if err != nil {
		return nil, fmt.Errorf("noise setup: %s", err)
	}
	localID := clientLink.chans.add(channel)

	msg, err := channel.HandshakeMessage()
	if err != nil {
		return nil, fmt.Errorf("noise handshake: %s", err)
	}

	if _, err := clientLink.link.SendOpenChannel(localID, remote, msg); err != nil {
		return nil, fmt.Errorf("failed to open channel: %s", err)
	}

	select {
	case result := <-channel.waitChan:
		if !result {
			return nil, ErrRejected
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	msgFactory.Logger.Infof("Opened new channel local ID %d, remote ID %d with %s", localID, channel.ID(), remote)
	return channel, nil
}

// Local returns the local public key.
func (msgFactory *MsgFactory) Local() cipher.PubKey {
	return msgFactory.pubKey
}

// Type returns the Transport type.
func (msgFactory *MsgFactory) Type() string {
	return "messaging"
}

// Close closes underlying link pool.
func (msgFactory *MsgFactory) Close() error {
	msgFactory.Logger.Info("Closing link pool")
	select {
	case <-msgFactory.doneCh:
	default:
		close(msgFactory.doneCh)
		msgFactory.newWG.Wait() // Ensure that 'c.newCh' is not being written to before closing.
		close(msgFactory.newCh)
	}
	return msgFactory.pool.Close()
}

func (msgFactory *MsgFactory) setEntry(ctx context.Context) error {
	msgFactory.Logger.Info("Updating discovery entry")
	serverPKs := []cipher.PubKey{}
	msgFactory.mu.RLock()
	for pk := range msgFactory.links {
		serverPKs = append(serverPKs, pk)
	}
	msgFactory.mu.RUnlock()

	entry, err := msgFactory.dc.Entry(ctx, msgFactory.pubKey)
	if err != nil {
		entry = client.NewClientEntry(msgFactory.pubKey, 0, serverPKs)
		if err := entry.Sign(msgFactory.secKey); err != nil {
			return err
		}

		return msgFactory.dc.SetEntry(ctx, entry)
	}

	entry.Client.DelegatedServers = serverPKs
	return msgFactory.dc.UpdateEntry(ctx, msgFactory.secKey, entry)
}

func (msgFactory *MsgFactory) link(remotePK cipher.PubKey, addr string) (*clientLink, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	l, err := msgFactory.pool.Initiate(conn, remotePK)
	if err != nil && err != ErrConnExists {
		return nil, err
	}

	msgFactory.Logger.Infof("Opened new link with the server %s", remotePK)
	clientLink := &clientLink{l, addr, newChanList()}
	msgFactory.mu.Lock()
	msgFactory.links[remotePK] = clientLink
	msgFactory.mu.Unlock()
	return clientLink, nil
}

func (msgFactory *MsgFactory) ensureLink(ctx context.Context, serverList []cipher.PubKey) (*clientLink, error) {
	for _, serverPK := range serverList {
		if l := msgFactory.getLink(serverPK); l != nil {
			return l, nil
		}
	}

	serverPK := serverList[0]
	serverEntry, err := msgFactory.dc.Entry(ctx, serverPK)
	if err != nil {
		return nil, fmt.Errorf("get server failure: %s", err)
	}

	l, err := msgFactory.link(serverPK, serverEntry.Server.Address)
	if err != nil {
		return nil, err
	}

	if err := msgFactory.setEntry(ctx); err != nil {
		return nil, fmt.Errorf("entry update failure: %s", err)
	}

	return l, nil
}

func (msgFactory *MsgFactory) getLink(remotePK cipher.PubKey) *clientLink {
	msgFactory.mu.RLock()
	l := msgFactory.links[remotePK]
	msgFactory.mu.RUnlock()
	return l
}

func (msgFactory *MsgFactory) onData(l *Link, frameType FrameType, body []byte) error {
	remotePK := l.Remote()
	if len(body) == 0 {
		msgFactory.Logger.Warnf("Invalid packet from %s: empty body", remotePK)
		return nil
	}

	clientLink := msgFactory.getLink(l.Remote())
	channelID := body[0]
	var sendErr error

	msgFactory.Logger.Debugf("New frame %s from %s@%d", frameType, remotePK, channelID)
	if frameType == FrameTypeOpenChannel {
		if lID, msg, err := msgFactory.openChannel(channelID, body[1:34], body[34:], clientLink); err != nil {
			msgFactory.Logger.Warnf("Failed to open new channel for %s: %s", remotePK, err)
			_, sendErr = l.SendChannelClosed(channelID)
		} else {
			msgFactory.Logger.Infof("Opened new channel local ID %d, remote ID %d with %s", lID, channelID,
				hex.EncodeToString(body[1:34]))
			_, sendErr = l.SendChannelOpened(channelID, lID, msg)
		}

		return msgFactory.warnSendError(remotePK, sendErr)
	}

	channel := clientLink.chans.get(channelID)
	if channel == nil {
		if frameType != FrameTypeChannelClosed && frameType != FrameTypeCloseChannel {
			msgFactory.Logger.Warnf("Frame for unknown channel %d from %s", channelID, remotePK)
		}
		return nil
	}

	switch frameType {
	case FrameTypeCloseChannel:
		clientLink.chans.remove(channelID)
		_, sendErr = l.SendChannelClosed(channel.ID())
		msgFactory.Logger.Debugf("Closed channel ID %d", channelID)
	case FrameTypeChannelOpened:
		channel.SetID(body[1])
		if err := channel.ProcessMessage(body[2:]); err != nil {
			sendErr = fmt.Errorf("noise handshake: %s", err)
		}

		select {
		case channel.waitChan <- true:
		default:
		}
	case FrameTypeChannelClosed:
		channel.SetID(body[0])
		select {
		case channel.waitChan <- false:
		default:
		}
		channel.OnChannelClosed()
		clientLink.chans.remove(channelID)
	case FrameTypeSend:
		go func() {
			select {
			case <-msgFactory.doneCh:
			case <-channel.doneChan:
			case channel.readChan <- body[1:]:
			}
		}()
	}

	return msgFactory.warnSendError(remotePK, sendErr)
}

func (msgFactory *MsgFactory) onClose(l *Link, remote bool) {
	remotePK := l.Remote()

	msgFactory.mu.RLock()
	chanLink := msgFactory.links[remotePK]
	msgFactory.mu.RUnlock()

	for _, channel := range chanLink.chans.dropAll() {
		channel.close()
	}

	select {
	case <-msgFactory.doneCh:
	default:
		msgFactory.Logger.Infof("Disconnected from the server %s. Trying to re-connect...", remotePK)
		for attempt := 0; attempt < msgFactory.retries; attempt++ {
			if _, err := msgFactory.link(remotePK, chanLink.addr); err == nil {
				msgFactory.Logger.Infof("Re-connected to the server %s", remotePK)
				return
			}
			time.Sleep(msgFactory.retryDelay)
		}
	}

	msgFactory.Logger.Infof("Closing link with the server %s", remotePK)

	msgFactory.mu.Lock()
	delete(msgFactory.links, remotePK)
	msgFactory.mu.Unlock()

	if err := msgFactory.setEntry(context.Background()); err != nil {
		msgFactory.Logger.Warnf("Failed to update entry: %s", err)
	}
}

func (msgFactory *MsgFactory) openChannel(rID byte, remotePK []byte, noiseMsg []byte, chanLink *clientLink) (lID byte, noiseRes []byte, err error) {
	var pubKey cipher.PubKey
	pubKey, err = cipher.NewPubKey(remotePK)
	if err != nil {
		return
	}

	channel, err := newChannel(false, msgFactory.secKey, pubKey, chanLink.link)
	channel.SetID(rID)
	if err != nil {
		err = fmt.Errorf("noise setup: %s", err)
		return
	}

	if err = channel.ProcessMessage(noiseMsg); err != nil {
		err = fmt.Errorf("noise handshake: %s", err)
		return
	}

	lID = chanLink.chans.add(channel)

	msgFactory.newWG.Add(1) // Ensure that 'c.newCh' is not being written to before closing.
	go func() {
		select {
		case <-msgFactory.doneCh:
		case msgFactory.newCh <- channel:
		}
		msgFactory.newWG.Done()
	}()

	noiseRes, err = channel.HandshakeMessage()
	if err != nil {
		err = fmt.Errorf("noise handshake: %s", err)
		return
	}

	return lID, noiseRes, err
}

func (msgFactory *MsgFactory) warnSendError(remote cipher.PubKey, err error) error {
	if err != nil {
		msgFactory.Logger.Warnf("Failed to send frame to %s: %s", remote, err)
	}

	return nil
}
