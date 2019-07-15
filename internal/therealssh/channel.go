package therealssh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kr/pty"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
)

// Port reserved for SSH app
const Port = 2

// Debug enables debug messages.
var Debug = false

// SSHChannel defines communication channel parameters.
type SSHChannel struct {
	log *logging.Logger

	RemoteID   uint32
	RemoteAddr *app.Addr

	conn  net.Conn
	msgCh chan []byte

	session    *Session
	listenerMx sync.Mutex
	listener   *net.UnixListener

	dataChMx sync.Mutex
	dataCh   chan []byte

	doneOnce sync.Once
	done     chan struct{}
}

// OpenChannel constructs new SSHChannel with empty Session.
func OpenChannel(remoteID uint32, remoteAddr *app.Addr, conn net.Conn) *SSHChannel {
	return &SSHChannel{log: logging.MustGetLogger("ssh_channel"), RemoteID: remoteID, conn: conn,
		RemoteAddr: remoteAddr, msgCh: make(chan []byte), dataCh: make(chan []byte), done: make(chan struct{})}
}

// OpenClientChannel constructs new client SSHChannel with empty Session.
func OpenClientChannel(remoteID uint32, remotePK cipher.PubKey, conn net.Conn) *SSHChannel {
	ch := OpenChannel(remoteID, &app.Addr{PubKey: remotePK, Port: Port}, conn)
	return ch
}

// Send sends command message.
func (sshCh *SSHChannel) Send(cmd CommandType, payload []byte) error {
	data := appendU32([]byte{byte(cmd)}, sshCh.RemoteID)
	_, err := sshCh.conn.Write(append(data, payload...))
	return err
}

func (sshCh *SSHChannel) Read(p []byte) (int, error) {
	data, more := <-sshCh.dataCh
	if !more {
		return 0, io.EOF
	}

	return copy(p, data), nil
}

func (sshCh *SSHChannel) Write(p []byte) (n int, err error) {
	n = len(p)
	err = sshCh.Send(CmdChannelData, p)
	return
}

// Request sends request message and waits for response.
func (sshCh *SSHChannel) Request(requestType RequestType, payload []byte) ([]byte, error) {
	sshCh.log.Debugf("sending request %x", requestType)
	req := append([]byte{byte(requestType)}, payload...)

	if err := sshCh.Send(CmdChannelRequest, req); err != nil {
		return nil, fmt.Errorf("request failure: %s", err)
	}

	data := <-sshCh.msgCh
	if data[0] == ResponseFail {
		return nil, fmt.Errorf("request failure: %s", string(data[1:]))
	}

	return data[1:], nil
}

// Serve starts request handling loop.
func (sshCh *SSHChannel) Serve() error {
	for data := range sshCh.msgCh {
		var err error
		sshCh.log.Debugf("new request %x", data[0])
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
			err = sshCh.OpenPTY(u, size)
		case RequestShell:
			err = sshCh.Shell()
		case RequestExec:
			err = sshCh.Start(string(data[1:]))
		case RequestWindowChange:
			cols := binary.BigEndian.Uint32(data[1:])
			rows := binary.BigEndian.Uint32(data[5:])
			width := binary.BigEndian.Uint32(data[9:])
			height := binary.BigEndian.Uint32(data[13:])
			err = sshCh.WindowChange(&pty.Winsize{Cols: uint16(cols), Rows: uint16(rows), X: uint16(width), Y: uint16(height)})
		}

		var res []byte
		if err != nil {
			res = append([]byte{ResponseFail}, []byte(err.Error())...)
		} else {
			res = []byte{ResponseConfirm}
		}

		if err := sshCh.Send(CmdChannelResponse, res); err != nil {
			return fmt.Errorf("failed to respond: %s", err)
		}
	}

	return nil
}

// SocketPath returns unix socket location. This socket is normally
// used by the CLI to exchange PTY data with a client app.
func (sshCh *SSHChannel) SocketPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("therealsshd-%d", sshCh.RemoteID))
}

// ServeSocket starts socket handling loop.
func (sshCh *SSHChannel) ServeSocket() error {
	os.Remove(sshCh.SocketPath())
	sshCh.log.Debugf("waiting for new socket connections on: %s", sshCh.SocketPath())
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: sshCh.SocketPath(), Net: "unix"})
	if err != nil {
		return fmt.Errorf("failed to open unix socket: %s", err)
	}

	sshCh.listenerMx.Lock()
	sshCh.listener = l
	sshCh.listenerMx.Unlock()
	conn, err := l.AcceptUnix()
	if err != nil {
		return fmt.Errorf("failed to accept connection: %s", err)
	}

	sshCh.log.Debugln("got new socket connection")
	defer func() {
		conn.Close()
		sshCh.closeListener() //nolint:errcheck
		os.Remove(sshCh.SocketPath())
	}()

	go func() {
		if _, err := io.Copy(sshCh, conn); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			sshCh.log.Errorf("failed to write to server:", err)
			return
		}
	}()

	if _, err := io.Copy(conn, sshCh); err != nil {
		return fmt.Errorf("failed to write to client: %s", err)
	}

	return nil
}

// OpenPTY creates new PTY Session for the Channel.
func (sshCh *SSHChannel) OpenPTY(user *user.User, sz *pty.Winsize) (err error) {
	if sshCh.session != nil {
		return errors.New("session is already started")
	}

	sshCh.log.Debugf("starting new session for %s with %#v", user.Username, sz)
	sshCh.session, err = OpenSession(user, sz)
	if err != nil {
		sshCh.session = nil
		return
	}

	return
}

// Shell starts shell process on Channel's PTY session.
func (sshCh *SSHChannel) Shell() error {
	return sshCh.Start("shell")
}

// Start executes provided command on Channel's PTY session.
func (sshCh *SSHChannel) Start(command string) error {
	if sshCh.session == nil {
		return errors.New("session is not started")
	}

	go func() {
		if err := sshCh.serveSession(); err != nil {
			sshCh.log.Errorf("Session failure:", err)
		}
	}()

	sshCh.log.Debugf("starting new pty process %s", command)
	return sshCh.session.Start(command)
}

func (sshCh *SSHChannel) serveSession() error {
	defer func() {
		sshCh.Send(CmdChannelServerClose, nil) // nolint
		sshCh.Close()
	}()

	go func() {
		if _, err := io.Copy(sshCh.session, sshCh); err != nil {
			sshCh.log.Errorf("PTY copy: ", err)
			return
		}
	}()

	_, err := io.Copy(sshCh, sshCh.session)
	if err != nil && !strings.Contains(err.Error(), "input/output error") {
		return fmt.Errorf("client copy: %s", err)
	}

	return nil
}

// WindowChange resize PTY Session size.
func (sshCh *SSHChannel) WindowChange(sz *pty.Winsize) error {
	if sshCh.session == nil {
		return errors.New("session is not started")
	}

	return sshCh.session.WindowChange(sz)
}

func (sshCh *SSHChannel) close() (closed bool, err error) {
	sshCh.doneOnce.Do(func() {
		closed = true

		close(sshCh.done)

		select {
		case <-sshCh.dataCh:
		default:
			sshCh.dataChMx.Lock()
			close(sshCh.dataCh)
			sshCh.dataChMx.Unlock()
		}
		close(sshCh.msgCh)

		var sErr, lErr error
		if sshCh.session != nil {
			sErr = sshCh.session.Close()
		}

		lErr = sshCh.closeListener()

		if sErr != nil {
			err = sErr
			return
		}

		if lErr != nil {
			err = lErr
		}
	})

	return closed, err
}

// Close safely closes Channel resources.
func (sshCh *SSHChannel) Close() error {
	if sshCh == nil {
		return nil
	}

	closed, err := sshCh.close()
	if err != nil {
		return err
	}
	if !closed {
		return errors.New("channel is already closed")
	}

	return nil
}

// IsClosed returns whether the Channel is closed.
func (sshCh *SSHChannel) IsClosed() bool {
	select {
	case <-sshCh.done:
		return true
	default:
		return false
	}
}

func (sshCh *SSHChannel) closeListener() error {
	sshCh.listenerMx.Lock()
	defer sshCh.listenerMx.Unlock()

	return sshCh.listener.Close()
}

func appendU32(buf []byte, n uint32) []byte {
	uintBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(uintBuf[0:], n)
	return append(buf, uintBuf...)
}
