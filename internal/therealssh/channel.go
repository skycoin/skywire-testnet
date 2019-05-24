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

	"github.com/kr/pty"

	"github.com/skycoin/skywire/pkg/cipher"

	"github.com/skycoin/skywire/pkg/app"
)

// Port reserved for therealssh app
const Port = 2

// Debug enables debug messages.
var Debug = false

// SshChannel defines communication channel parameters.
type SshChannel struct {
	RemoteID   uint32
	RemoteAddr *app.Addr

	conn  net.Conn
	msgCh chan []byte

	session  *Session
	listener *net.UnixListener
	dataCh   chan []byte
}

// OpenChannel constructs new SshChannel with empty Session.
func OpenChannel(remoteID uint32, remoteAddr *app.Addr, conn net.Conn) *SshChannel {
	return &SshChannel{RemoteID: remoteID, conn: conn, RemoteAddr: remoteAddr, msgCh: make(chan []byte), dataCh: make(chan []byte)}
}

// OpenClientChannel constructs new client SshChannel with empty Session.
func OpenClientChannel(remoteID uint32, remotePK cipher.PubKey, conn net.Conn) *SshChannel {
	ch := OpenChannel(remoteID, &app.Addr{PubKey: remotePK, Port: Port}, conn)
	return ch
}

// Send sends command message.
func (sshCh *SshChannel) Send(cmd CommandType, payload []byte) error {
	data := appendU32([]byte{byte(cmd)}, sshCh.RemoteID)
	_, err := sshCh.conn.Write(append(data, payload...))
	return err
}

func (sshCh *SshChannel) Read(p []byte) (int, error) {
	data, more := <-sshCh.dataCh
	if !more {
		return 0, io.EOF
	}

	return copy(p, data), nil
}

func (sshCh *SshChannel) Write(p []byte) (n int, err error) {
	n = len(p)
	err = sshCh.Send(CmdChannelData, p)
	return
}

// Request sends request message and waits for response.
func (sshCh *SshChannel) Request(requestType RequestType, payload []byte) ([]byte, error) {
	debug("sending request %x", requestType)
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
func (sshCh *SshChannel) Serve() error {
	for data := range sshCh.msgCh {
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
func (sshCh *SshChannel) SocketPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("therealsshd-%d", sshCh.RemoteID))
}

// ServeSocket starts socket handling loop.
func (sshCh *SshChannel) ServeSocket() error {
	os.Remove(sshCh.SocketPath())
	debug("waiting for new socket connections on: %s", sshCh.SocketPath())
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: sshCh.SocketPath(), Net: "unix"})
	if err != nil {
		return fmt.Errorf("failed to open unix socket: %s", err)
	}

	sshCh.listener = l
	conn, err := l.AcceptUnix()
	if err != nil {
		return fmt.Errorf("failed to accept connection: %s", err)
	}

	debug("got new socket connection")
	defer func() {
		conn.Close()
		sshCh.listener.Close()
		sshCh.listener = nil
		os.Remove(sshCh.SocketPath())
	}()

	go func() {
		if _, err := io.Copy(sshCh, conn); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			log.Println("failed to write to server:", err)
			return
		}
	}()

	if _, err := io.Copy(conn, sshCh); err != nil {
		return fmt.Errorf("failed to write to client: %s", err)
	}

	return nil
}

// OpenPTY creates new PTY Session for the Channel.
func (sshCh *SshChannel) OpenPTY(user *user.User, sz *pty.Winsize) (err error) {
	if sshCh.session != nil {
		return errors.New("session is already started")
	}

	debug("starting new session for %s with %#v", user.Username, sz)
	sshCh.session, err = OpenSession(user, sz)
	if err != nil {
		sshCh.session = nil
		return
	}

	return
}

// Shell starts shell process on Channel's PTY session.
func (sshCh *SshChannel) Shell() error {
	return sshCh.Start("shell")
}

// Start executes provided command on Channel's PTY session.
func (sshCh *SshChannel) Start(command string) error {
	if sshCh.session == nil {
		return errors.New("session is not started")
	}

	go func() {
		if err := sshCh.serveSession(); err != nil {
			log.Println("Session failure:", err)
		}
	}()

	debug("starting new pty process %s", command)
	return sshCh.session.Start(command)
}

func (sshCh *SshChannel) serveSession() error {
	defer func() {
		sshCh.Send(CmdChannelServerClose, nil) // nolint
		sshCh.Close()
	}()

	go func() {
		if _, err := io.Copy(sshCh.session, sshCh); err != nil {
			log.Println("PTY copy: ", err)
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
func (sshCh *SshChannel) WindowChange(sz *pty.Winsize) error {
	if sshCh.session == nil {
		return errors.New("session is not started")
	}

	return sshCh.session.WindowChange(sz)
}

// Close safely closes Channel resources.
func (sshCh *SshChannel) Close() error {
	select {
	case <-sshCh.dataCh:
	default:
		close(sshCh.dataCh)
	}
	close(sshCh.msgCh)

	var sErr, lErr error
	if sshCh.session != nil {
		sErr = sshCh.session.Close()
	}

	if sshCh.listener != nil {
		lErr = sshCh.listener.Close()
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
