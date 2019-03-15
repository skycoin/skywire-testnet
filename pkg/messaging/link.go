package messaging

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/cipher"
)

const (
	// DefaultHandshakeTimeout represents the default handshake timeout.
	DefaultHandshakeTimeout = time.Second * 3
)

// LinkConfig represents the common config of a connection.
type LinkConfig struct {
	Public           cipher.PubKey
	Secret           cipher.SecKey
	Remote           cipher.PubKey // Public static key of the remote instance.
	HandshakeTimeout time.Duration
	Logger           *logging.Logger
	Initiator        bool // Whether the local instance initiates the connection.
}

// DefaultLinkConfig returns a connection configuration with default values.
func DefaultLinkConfig() *LinkConfig {
	return &LinkConfig{
		HandshakeTimeout: DefaultHandshakeTimeout,
		Logger:           logging.MustGetLogger("link"),
	}
}

// Link represents a messaging connection in the perspective of an instance.
type Link struct {
	rw        io.ReadWriteCloser
	config    *LinkConfig
	callbacks *Callbacks
}

// NewLink creates a new Link.
func NewLink(rw io.ReadWriteCloser, config *LinkConfig, callbacks *Callbacks) (*Link, error) {
	return &Link{
		rw:        rw,
		config:    config,
		callbacks: callbacks,
	}, nil
}

func (c *Link) logf(format string, a ...interface{}) {
	if c.config.Logger != nil {
		prefix := fmt.Sprintf("[%s::%s]",
			c.config.Public,
			c.config.Remote)
		c.config.Logger.Info(prefix, fmt.Sprintf(format, a...))
	}
}

// Open performs a handshake with the remote instance and attempts to establish a connection.
func (c *Link) Open(wg *sync.WaitGroup) error {
	var handshake Handshake
	if c.config.Initiator {
		handshake = initiatorHandshake(c.config)
	} else {
		handshake = responderHandshake(c.config)
	}

	// Perform handshake.
	if err := handshake.Do(json.NewDecoder(c.rw), json.NewEncoder(c.rw), c.config.HandshakeTimeout); err != nil {
		return err
	}

	// Handshake complete callback.
	c.callbacks.HandshakeComplete(c)

	// Event loops.
	var done = make(chan struct{})
	wg.Add(1)
	go func() {
		// Exits when connection is closed.
		if err := c.readLoop(); err != nil {
			c.logf("CLOSED: err(%v)", err)
		}
		// TODO(evanlinjin): Determine if the 'close' is initiated from remote instance.
		c.callbacks.Close(c, false)
		close(done)
		wg.Done()
	}()

	return nil
}

// Close closes the connection with the remote instance.
func (c *Link) Close() error {
	return c.rw.Close()
}

// SendOpenChannel sends OpenChannel request.
func (c *Link) SendOpenChannel(channelID byte, remotePK cipher.PubKey, noiseMsg []byte) (int, error) {
	payload := append([]byte{channelID}, remotePK[:]...)
	return c.writeFrame(FrameTypeOpenChannel, append(payload, noiseMsg...))
}

// SendChannelOpened sends ChannelOpened frame.
func (c *Link) SendChannelOpened(channelID byte, remoteID byte, noiseMsg []byte) (int, error) {
	return c.writeFrame(FrameTypeChannelOpened, append([]byte{channelID, remoteID}, noiseMsg...))
}

// SendCloseChannel sends CloseChannel request.
func (c *Link) SendCloseChannel(channelID byte) (int, error) {
	return c.writeFrame(FrameTypeCloseChannel, []byte{channelID})
}

// SendChannelClosed sends ChannelClosed frame.
func (c *Link) SendChannelClosed(channelID byte) (int, error) {
	return c.writeFrame(FrameTypeChannelClosed, []byte{channelID})
}

// Send sends data frame.
func (c *Link) Send(channelID byte, body []byte) (int, error) {
	return c.writeFrame(FrameTypeSend, append([]byte{channelID}, body...))
}

// Config returns the instance's read-only configuration.
func (c *Link) Config() *LinkConfig {
	return c.config
}

// Local returns the local PubKey.
func (c *Link) Local() cipher.PubKey {
	return c.config.Public
}

// Remote returns the remote PubKey.
func (c *Link) Remote() cipher.PubKey {
	return c.config.Remote
}

// Initiator returns whether the current instance is the initiator.
func (c *Link) Initiator() bool {
	return c.config.Initiator
}

// event loop that processing packets
// runs after handshake
func (c *Link) readLoop() error {
	for {
		switch err := c.handleData(ReadFrame(c.rw)); err {
		case nil:
			continue
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

func (c *Link) handleData(payload Frame, n int, err error) error {
	if err != nil {
		return err
	}
	t, b := payload.Type(), payload.Body()
	c.logf("RECEIVED: type(%d) bytes(%d)", t, n)
	if err := c.readFrame(t, b); err != nil {
		return err
	}
	return nil
}

func (c *Link) writeFrame(t FrameType, body []byte) (int, error) {
	payload := MakeFrame(t, body)
	n, err := WriteFrame(c.rw, payload)
	if err != nil {
		return 0, err
	}
	c.logf("    SENT: type(%d) bytes(%d)", t, n)
	return n, nil
}

func (c *Link) readFrame(dt FrameType, body []byte) error {
	return c.callbacks.Data(c, dt, body)
}
