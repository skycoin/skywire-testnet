package therealssh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/skycoin/skywire/pkg/app"

	"github.com/kr/pty"

	"github.com/skycoin/skywire/pkg/cipher"
)

// Port reserved for therealssh app
const Port = 2

// Debug enables debug messages.
var Debug = false

// Channel defines communication channel parameters.
type Channel struct {
	RemoteID   uint32
	RemoteAddr *app.LoopAddr

	conn  net.Conn
	msgCh chan []byte

	session  *Session
	listener *net.UnixListener
	dataCh   chan []byte
}

// OpenChannel constructs new Channel with empty Session.
func OpenChannel(remoteID uint32, remoteAddr *app.LoopAddr, conn net.Conn) *Channel {
	return &Channel{RemoteID: remoteID, conn: conn, RemoteAddr: remoteAddr, msgCh: make(chan []byte), dataCh: make(chan []byte)}
}

// OpenClientChannel constructs new client Channel with empty Session.
func OpenClientChannel(remoteID uint32, remotePK cipher.PubKey, conn net.Conn) *Channel {
	ch := OpenChannel(remoteID, &app.LoopAddr{PubKey: remotePK, Port: Port}, conn)
	return ch
}

// Send sends command message.
func (c *Channel) Send(cmd CommandType, payload []byte) error {
	data := appendU32([]byte{byte(cmd)}, c.RemoteID)
	_, err := c.conn.Write(append(data, payload...))
	return err
}

func (c *Channel) Read(p []byte) (int, error) {
	data, more := <-c.dataCh
	if !more {
		return 0, io.EOF
	}

	return copy(p, data), nil
}

func (c *Channel) Write(p []byte) (n int, err error) {
	n = len(p)
	err = c.Send(CmdChannelData, p)
	return
}

// Request sends request message and waits for response.
func (c *Channel) Request(requestType RequestType, payload []byte) ([]byte, error) {
	debug("sending request %x", requestType)
	req := append([]byte{byte(requestType)}, payload...)

	if err := c.Send(CmdChannelRequest, req); err != nil {
		return nil, fmt.Errorf("request failure: %s", err)
	}

	data := <-c.msgCh
	if data[0] == ResponseFail {
		return nil, fmt.Errorf("request failure: %s", string(data[1:]))
	}

	return data[1:], nil
}

// Serve starts request handling loop.
func (c *Channel) Serve() error {
	for data := range c.msgCh {
		var err error
		debug("new request %x", data[0])
		switch RequestType(data[0]) {
		case RequestPTY:
			var u *user.User
			u, err = user.Lookup(string(data[17:]))
			if err != nil {
				break
			}

			cols := binary.BigEndian.Uint32(data[1:])
			rows := binary.BigEndian.Uint32(data[5:])
			width := binary.BigEndian.Uint32(data[9:])
			height := binary.BigEndian.Uint32(data[13:])
			size := &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows), X: uint16(width), Y: uint16(height)}
			err = c.OpenPTY(u, size)
		case RequestShell:
			err = c.Shell()
		case RequestExec:
			err = c.Start(string(data[1:]))
		case RequestWindowChange:
			cols := binary.BigEndian.Uint32(data[1:])
			rows := binary.BigEndian.Uint32(data[5:])
			width := binary.BigEndian.Uint32(data[9:])
			height := binary.BigEndian.Uint32(data[13:])
			err = c.WindowChange(&pty.Winsize{Cols: uint16(cols), Rows: uint16(rows), X: uint16(width), Y: uint16(height)})
		}

		var res []byte
		if err != nil {
			res = append([]byte{ResponseFail}, []byte(err.Error())...)
		} else {
			res = []byte{ResponseConfirm}
		}

		if err := c.Send(CmdChannelResponse, res); err != nil {
			return fmt.Errorf("failed to respond: %s", err)
		}
	}

	return nil
}

// SocketPath returns unix socket location. This socket is normally
// used by the CLI to exchange PTY data with a client app.
func (c *Channel) SocketPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("therealsshd-%d", c.RemoteID))
}

// ServeSocket starts socket handling loop.
func (c *Channel) ServeSocket() error {
	os.Remove(c.SocketPath())
	debug("waiting for new socket connections on: %s", c.SocketPath())
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: c.SocketPath(), Net: "unix"})
	if err != nil {
		return fmt.Errorf("failed to open unix socket: %s", err)
	}

	c.listener = l
	conn, err := l.AcceptUnix()
	if err != nil {
		return fmt.Errorf("failed to accept connection: %s", err)
	}

	debug("got new socket connection")
	defer func() {
		conn.Close()
		c.listener.Close()
		c.listener = nil
		os.Remove(c.SocketPath())
	}()

	go func() {
		if _, err := io.Copy(c, conn); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			log.Println("failed to write to server:", err)
			return
		}
	}()

	if _, err := io.Copy(conn, c); err != nil {
		return fmt.Errorf("failed to write to client: %s", err)
	}

	return nil
}

// OpenPTY creates new PTY Session for the Channel.
func (c *Channel) OpenPTY(user *user.User, sz *pty.Winsize) (err error) {
	if c.session != nil {
		return errors.New("session is already started")
	}

	debug("starting new session for %s with %#v", user.Username, sz)
	c.session, err = OpenSession(user, sz)
	if err != nil {
		c.session = nil
		return
	}

	return
}

// Shell starts shell process on Channel's PTY session.
func (c *Channel) Shell() error {
	return c.Start("shell")
}

// Start executes provided command on Channel's PTY session.
func (c *Channel) Start(command string) error {
	if c.session == nil {
		return errors.New("session is not started")
	}

	go func() {
		if err := c.serveSession(); err != nil {
			log.Println("Session failure:", err)
		}
	}()

	debug("starting new pty process %s", command)
	return c.session.Start(command)
}

func (c *Channel) serveSession() error {
	defer func() {
		c.Send(CmdChannelServerClose, nil) // nolint
		c.Close()
	}()

	go func() {
		if _, err := io.Copy(c.session, c); err != nil {
			log.Println("PTY copy: ", err)
			return
		}
	}()

	_, err := io.Copy(c, c.session)
	if err != nil && !strings.Contains(err.Error(), "input/output error") {
		return fmt.Errorf("client copy: %s", err)
	}

	return nil
}

// WindowChange resize PTY Session size.
func (c *Channel) WindowChange(sz *pty.Winsize) error {
	if c.session == nil {
		return errors.New("session is not started")
	}

	return c.session.WindowChange(sz)
}

// Close safely closes Channel resources.
func (c *Channel) Close() error {
	select {
	case <-c.dataCh:
	default:
		close(c.dataCh)
	}
	close(c.msgCh)

	var sErr, lErr error
	if c.session != nil {
		sErr = c.session.Close()
	}

	if c.listener != nil {
		lErr = c.listener.Close()
	}

	if sErr != nil {
		return sErr
	}

	if lErr != nil {
		return lErr
	}

	return nil
}

func debug(format string, v ...interface{}) {
	if !Debug {
		return
	}

	log.Printf(format, v...)
}

func appendU32(buf []byte, n uint32) []byte {
	uintBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(uintBuf[0:], n)
	return append(buf, uintBuf...)
}
