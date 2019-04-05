// Package messaging implements messaging communication. Messaging
// communication is performed between 2 nodes using intermediate relay
// server, node discovery is performed using messaging discovery.
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
	ErrChannelClosed = errors.New("channel closed")
	// ErrDeadlineExceeded indicates that read/write operation failed due to timeout.
	ErrDeadlineExceeded = errors.New("deadline exceeded")
	// ErrClientClosed indicates that client is closed and not accepting new connections.
	ErrClientClosed = errors.New("client closed")
)

type clientLink struct {
	link  *Link
	addr  string
	chans *chanList
}

// Config configures Client.
type Config struct {
	PubKey     cipher.PubKey
	SecKey     cipher.SecKey
	Discovery  client.APIClient
	Retries    int
	RetryDelay time.Duration
}

// Client sends messages to remote client nodes via relay Server.
// Implements Transport
type Client struct {
	Logger *logging.Logger

	// edges [2]cipher.PubKey
	pubKey cipher.PubKey
	secKey cipher.SecKey
	dc     client.APIClient
	pool   *Pool

	retries    int
	retryDelay time.Duration

	links map[cipher.PubKey]*clientLink
	mu    sync.RWMutex

	newChan  chan *channel
	doneChan chan struct{}
}

// NewClient constructs a new Client.
func NewClient(conf *Config) *Client {
	c := &Client{
		Logger:     logging.MustGetLogger("messenger"),
		pubKey:     conf.PubKey,
		secKey:     conf.SecKey,
		dc:         conf.Discovery,
		retries:    conf.Retries,
		retryDelay: conf.RetryDelay,
		links:      make(map[cipher.PubKey]*clientLink),
		newChan:    make(chan *channel),
		doneChan:   make(chan struct{}),
	}
	config := &LinkConfig{
		Public:           c.pubKey,
		Secret:           c.secKey,
		HandshakeTimeout: DefaultHandshakeTimeout,
	}
	c.pool = NewPool(config, &Callbacks{
		Data:  c.onData,
		Close: c.onClose,
	})

	return c
}

// ConnectToInitialServers tries to connect to at most serverCount servers.
func (c *Client) ConnectToInitialServers(ctx context.Context, serverCount int) error {
	if serverCount == 0 {
		return nil
	}

	entries, err := c.dc.AvailableServers(ctx)
	if err != nil {
		return fmt.Errorf("servers are not available: %s", err)
	}

	for _, entry := range entries {
		if len(c.links) > serverCount {
			break
		}

		if _, err := c.link(entry.Static, entry.Server.Address); err != nil {
			c.Logger.Warnf("Failed to connect to the server %s: %s", entry.Static, err)
		}
	}

	if len(c.links) == 0 {
		return fmt.Errorf("servers are not available: all servers failed")
	}

	if err := c.setEntry(ctx); err != nil {
		return fmt.Errorf("entry update failure: %s", err)
	}

	return nil
}

// Accept accepts a remotely-initiated Transport.
func (c *Client) Accept(ctx context.Context) (transport.Transport, error) {
	select {
	case ch, more := <-c.newChan:
		if !more {
			return nil, ErrClientClosed
		}
		return newAckedChannel(ch), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Dial initiates a Transport with a remote node.
func (c *Client) Dial(ctx context.Context, remote cipher.PubKey) (transport.Transport, error) {
	entry, err := c.dc.Entry(ctx, remote)
	if err != nil {
		fmt.Printf("Dial with remote = %v\n", remote)
		return nil, fmt.Errorf("get entry failure: %s", err)
	}

	if entry.Client.DelegatedServers == nil || len(entry.Client.DelegatedServers) == 0 {
		return nil, ErrNoSrv
	}

	clientLink, err := c.ensureLink(ctx, entry.Client.DelegatedServers)
	if err != nil {
		return nil, fmt.Errorf("link failure: %s", err)
	}

	channel, err := newChannel(true, c.secKey, remote, clientLink.link)
	if err != nil {
		return nil, fmt.Errorf("noise setup: %s", err)
	}
	localID := clientLink.chans.add(channel)

	msg, err := channel.noise.HandshakeMessage()
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

	c.Logger.Infof("Opened new channel local ID %d, remote ID %d with %s", localID, channel.ID, remote)
	return newAckedChannel(channel), nil
}

// Local returns the local public key.
func (c *Client) Local() cipher.PubKey {
	return c.pubKey
}

// Type returns the Transport type.
func (c *Client) Type() string {
	return "messaging"
}

// Close closes underlying link pool.
func (c *Client) Close() error {
	c.Logger.Info("Closing link pool")
	select {
	case <-c.doneChan:
	default:
		close(c.doneChan)
		close(c.newChan)
	}
	return c.pool.Close()
}

func (c *Client) setEntry(ctx context.Context) error {
	c.Logger.Info("Updating discovery entry")
	serverPKs := []cipher.PubKey{}
	c.mu.RLock()
	for pk := range c.links {
		serverPKs = append(serverPKs, pk)
	}
	c.mu.RUnlock()

	entry, err := c.dc.Entry(ctx, c.pubKey)
	if err != nil {
		entry = client.NewClientEntry(c.pubKey, 0, serverPKs)
		if err := entry.Sign(c.secKey); err != nil {
			return err
		}

		return c.dc.SetEntry(ctx, entry)
	}

	entry.Client.DelegatedServers = serverPKs
	return c.dc.UpdateEntry(ctx, c.secKey, entry)
}

func (c *Client) link(remotePK cipher.PubKey, addr string) (*clientLink, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	l, err := c.pool.Initiate(conn, remotePK)
	if err != nil && err != ErrConnExists {
		return nil, err
	}

	c.Logger.Infof("Opened new link with the server %s", remotePK)
	clientLink := &clientLink{l, addr, newChanList()}
	c.mu.Lock()
	c.links[remotePK] = clientLink
	c.mu.Unlock()
	return clientLink, nil
}

func (c *Client) ensureLink(ctx context.Context, serverList []cipher.PubKey) (*clientLink, error) {
	for _, serverPK := range serverList {
		if l := c.getLink(serverPK); l != nil {
			return l, nil
		}
	}

	serverPK := serverList[0]
	serverEntry, err := c.dc.Entry(ctx, serverPK)
	if err != nil {
		return nil, fmt.Errorf("get server failure: %s", err)
	}

	l, err := c.link(serverPK, serverEntry.Server.Address)
	if err != nil {
		return nil, err
	}

	if err := c.setEntry(ctx); err != nil {
		return nil, fmt.Errorf("entry update failure: %s", err)
	}

	return l, nil
}

func (c *Client) getLink(remotePK cipher.PubKey) *clientLink {
	c.mu.RLock()
	l := c.links[remotePK]
	c.mu.RUnlock()
	return l
}

func (c *Client) onData(l *Link, frameType FrameType, body []byte) error {
	remotePK := l.Remote()
	if len(body) == 0 {
		c.Logger.Warnf("Invalid packet from %s: empty body", remotePK)
		return nil
	}

	clientLink := c.getLink(l.Remote())
	channelID := body[0]
	var sendErr error

	c.Logger.Debugf("New frame %s from %s@%d", frameType, remotePK, channelID)
	if frameType == FrameTypeOpenChannel {
		if lID, msg, err := c.openChannel(channelID, body[1:34], body[34:], clientLink); err != nil {
			c.Logger.Warnf("Failed to open new channel for %s: %s", remotePK, err)
			_, sendErr = l.SendChannelClosed(channelID)
		} else {
			c.Logger.Infof("Opened new channel local ID %d, remote ID %d with %s", lID, channelID,
				hex.EncodeToString(body[1:34]))
			_, sendErr = l.SendChannelOpened(channelID, lID, msg)
		}

		return c.warnSendError(remotePK, sendErr)
	}

	channel := clientLink.chans.get(channelID)
	if channel == nil {
		if frameType != FrameTypeChannelClosed && frameType != FrameTypeCloseChannel {
			c.Logger.Warnf("Frame for unknown channel %d from %s", channelID, remotePK)
		}
		return nil
	}

	switch frameType {
	case FrameTypeCloseChannel:
		clientLink.chans.remove(channelID)
		_, sendErr = l.SendChannelClosed(channel.ID)
		c.Logger.Debugf("Closed channel ID %d", channelID)
	case FrameTypeChannelOpened:
		channel.ID = body[1]
		if err := channel.noise.ProcessMessage(body[2:]); err != nil {
			sendErr = fmt.Errorf("noise handshake: %s", err)
		}

		select {
		case channel.waitChan <- true:
		default:
		}
	case FrameTypeChannelClosed:
		channel.ID = body[0]
		select {
		case channel.waitChan <- false:
		case channel.closeChan <- struct{}{}:
			clientLink.chans.remove(channelID)
		default:
		}
	case FrameTypeSend:
		go func() {
			select {
			case <-c.doneChan:
			case <-channel.doneChan:
			case channel.readChan <- body[1:]:
			}
		}()
	}

	return c.warnSendError(remotePK, sendErr)
}

func (c *Client) onClose(l *Link, remote bool) {
	remotePK := l.Remote()

	c.mu.RLock()
	chanLink := c.links[remotePK]
	c.mu.RUnlock()

	for _, channel := range chanLink.chans.dropAll() {
		channel.close()
	}

	select {
	case <-c.doneChan:
	default:
		c.Logger.Infof("Disconnected from the server %s. Trying to re-connect...", remotePK)
		for attemp := 0; attemp < c.retries; attemp++ {
			if _, err := c.link(remotePK, chanLink.addr); err == nil {
				c.Logger.Infof("Re-connected to the server %s", remotePK)
				return
			}
			time.Sleep(c.retryDelay)
		}
	}

	c.Logger.Infof("Closing link with the server %s", remotePK)

	c.mu.Lock()
	delete(c.links, remotePK)
	c.mu.Unlock()

	if err := c.setEntry(context.Background()); err != nil {
		c.Logger.Warnf("Failed to update entry: %s", err)
	}
}

func (c *Client) openChannel(rID byte, remotePK []byte, noiseMsg []byte, chanLink *clientLink) (lID byte, noiseRes []byte, err error) {
	var pubKey cipher.PubKey
	pubKey, err = cipher.NewPubKey(remotePK)
	if err != nil {
		return
	}

	channel, err := newChannel(false, c.secKey, pubKey, chanLink.link)
	channel.ID = rID
	if err != nil {
		err = fmt.Errorf("noise setup: %s", err)
		return
	}

	if err = channel.noise.ProcessMessage(noiseMsg); err != nil {
		err = fmt.Errorf("noise handshake: %s", err)
		return
	}

	lID = chanLink.chans.add(channel)

	go func() {
		select {
		case <-c.doneChan:
		case c.newChan <- channel:
		}
	}()

	noiseRes, err = channel.noise.HandshakeMessage()
	if err != nil {
		err = fmt.Errorf("noise handshake: %s", err)
		return
	}

	return lID, noiseRes, err
}

func (c *Client) warnSendError(remote cipher.PubKey, err error) error {
	if err != nil {
		c.Logger.Warnf("Failed to send frame to %s: %s", remote, err)
	}

	return nil
}
