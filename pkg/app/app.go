/*
Package app implements app to node communication interface.
*/
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"
)

const (
	// DefaultIn holds value of inFd for Apps setup via Node
	DefaultIn = uintptr(3)

	// DefaultOut holds value of outFd for Apps setup via Node
	DefaultOut = uintptr(4)
)

var (
	log = logging.MustGetLogger("app")
)

// Config defines configuration parameters for App
type Config struct {
	AppName         string `json:"app-name"`
	AppVersion      string `json:"app-version"`
	ProtocolVersion string `json:"protocol-version"`
}

// App represents client side in app's client-server communication
// interface.
type App struct {
	config Config
	proto  *Protocol

	acceptChan chan [2]*Addr
	doneChan   chan struct{}

	conns map[LoopAddr]io.ReadWriteCloser
	mu    sync.Mutex
}

// Command setups pipe connection and returns *exec.Cmd for an App
// with initialized connection.
func Command(config *Config, appsPath string, args []string) (net.Conn, *exec.Cmd, error) {
	srvConn, clientConn, err := OpenPipeConn()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open piped connection: %s", err)
	}

	binaryPath := filepath.Join(appsPath, fmt.Sprintf("%s.v%s", config.AppName, config.AppVersion))
	cmd := exec.Command(binaryPath, args...) // nolint:gosec
	cmd.ExtraFiles = []*os.File{clientConn.inFile, clientConn.outFile}

	return srvConn, cmd, nil
}

// SetupFromPipe connects to a pipe, starts protocol loop and performs
// initialization request with the Server.
func SetupFromPipe(config *Config, inFD, outFD uintptr) (*App, error) {
	pipeConn, err := NewPipeConn(inFD, outFD)
	if err != nil {
		return nil, fmt.Errorf("failed to open pipe: %s", err)
	}

	app := &App{
		config:     *config,
		proto:      NewProtocol(pipeConn),
		acceptChan: make(chan [2]*Addr),
		doneChan:   make(chan struct{}),
		conns:      make(map[LoopAddr]io.ReadWriteCloser),
	}

	go app.handleProto()

	if err := app.proto.Send(FrameInit, config, nil); err != nil {
		if err := app.Close(); err != nil {
			log.WithError(err).Warn("Failed to close app")
		}
		return nil, fmt.Errorf("INIT handshake failed: %s", err)
	}

	return app, nil
}

// Setup setups app using default pair of pipes
func Setup(config *Config) (*App, error) {
	return SetupFromPipe(config, DefaultIn, DefaultOut)
}

// Close implements io.Closer for an App.
func (app *App) Close() error {
	if app == nil {
		return nil
	}

	select {
	case <-app.doneChan: // already closed
	default:
		close(app.doneChan)
	}

	app.mu.Lock()
	for addr, conn := range app.conns {
		if err := app.proto.Send(FrameClose, &addr, nil); err != nil {
			log.WithError(err).Warn("Failed to send command frame")
		}
		if err := conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close connection")
		}
	}
	app.mu.Unlock()

	return app.proto.Close()
}

// Accept awaits for incoming loop confirmation request from a Node and
// returns net.Conn for received loop.
func (app *App) Accept() (net.Conn, error) {
	addrs := <-app.acceptChan
	laddr := addrs[0]
	raddr := addrs[1]

	addr := &LoopAddr{laddr.Port, *raddr}
	conn, out := net.Pipe()
	app.mu.Lock()
	app.conns[*addr] = conn
	app.mu.Unlock()
	go app.serveConn(addr, conn)
	return newAppConn(out, laddr, raddr), nil
}

// Dial sends create loop request to a Node and returns net.Conn for created loop.
func (app *App) Dial(raddr *Addr) (net.Conn, error) {
	laddr := &Addr{}
	err := app.proto.Send(FrameCreateLoop, raddr, laddr)
	if err != nil {
		return nil, err
	}
	addr := &LoopAddr{laddr.Port, *raddr}
	conn, out := net.Pipe()
	app.mu.Lock()
	app.conns[*addr] = conn
	app.mu.Unlock()
	go app.serveConn(addr, conn)
	return newAppConn(out, laddr, raddr), nil
}

// Addr returns empty Addr, implements net.Listener.
func (app *App) Addr() net.Addr {
	return &Addr{}
}

func (app *App) handleProto() {
	err := app.proto.Serve(func(frame Frame, payload []byte) (res interface{}, err error) {
		switch frame {
		case FrameConfirmLoop:
			err = app.confirmLoop(payload)
		case FrameSend:
			err = app.forwardPacket(payload)
		case FrameClose:
			err = app.closeConn(payload)
		default:
			err = errors.New("unexpected frame")
		}

		return res, err
	})

	if err != nil {
		return
	}
}

func (app *App) serveConn(addr *LoopAddr, conn io.ReadWriteCloser) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.WithError(err).Warn("failed to close connection")
		}
	}()

	for {
		buf := make([]byte, 32*1024)
		n, err := conn.Read(buf)
		if err != nil {
			break
		}

		packet := &Packet{Addr: addr, Payload: buf[:n]}
		if err := app.proto.Send(FrameSend, packet, nil); err != nil {
			break
		}
	}

	app.mu.Lock()
	if _, ok := app.conns[*addr]; ok {
		if err := app.proto.Send(FrameClose, &addr, nil); err != nil {
			log.WithError(err).Warn("Failed to send command frame")
		}
	}
	delete(app.conns, *addr)
	app.mu.Unlock()
}

func (app *App) forwardPacket(data []byte) error {
	packet := &Packet{}
	if err := json.Unmarshal(data, packet); err != nil {
		return err
	}

	app.mu.Lock()
	conn := app.conns[*packet.Addr]
	app.mu.Unlock()

	if conn == nil {
		return errors.New("no listeners")
	}

	_, err := conn.Write(packet.Payload)
	return err
}

func (app *App) closeConn(data []byte) error {
	addr := &LoopAddr{}
	if err := json.Unmarshal(data, addr); err != nil {
		return err
	}

	app.mu.Lock()
	conn := app.conns[*addr]
	delete(app.conns, *addr)
	app.mu.Unlock()

	return conn.Close()
}

func (app *App) confirmLoop(data []byte) error {
	addrs := [2]*Addr{}
	if err := json.Unmarshal(data, &addrs); err != nil {
		return err
	}

	laddr := addrs[0]
	raddr := addrs[1]

	app.mu.Lock()
	conn := app.conns[LoopAddr{laddr.Port, *raddr}]
	app.mu.Unlock()

	if conn != nil {
		return errors.New("loop is already created")
	}

	select {
	case app.acceptChan <- addrs:
	default:
	}

	return nil
}

type appConn struct {
	net.Conn
	laddr *Addr
	raddr *Addr
}

func newAppConn(conn net.Conn, laddr, raddr *Addr) *appConn {
	return &appConn{conn, laddr, raddr}
}

func (conn *appConn) LocalAddr() net.Addr {
	return conn.laddr
}

func (conn *appConn) RemoteAddr() net.Addr {
	return conn.raddr
}
