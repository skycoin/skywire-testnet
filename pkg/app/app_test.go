package app

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/routing"
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

func TestAppDial(t *testing.T) {
	lpk, _ := cipher.GenerateKeyPair()
	rpk, _ := cipher.GenerateKeyPair()

	in, out := net.Pipe()
	proto := NewProtocol(out)
	app := &App{proto: NewProtocol(in), conns: make(map[LoopAddr]io.ReadWriteCloser)}
	go app.handleProto()

	dataCh := make(chan []byte)
	go proto.Serve(func(f Frame, p []byte) (interface{}, error) { // nolint: errcheck
		if f == FrameCreateLoop {
			return &routing.Addr{PubKey: lpk, Port: 2}, nil
		}

		if f == FrameClose {
			go func() { dataCh <- p }()
			return nil, nil
		}

		return nil, errors.New("unexpected frame")
	})
	conn, err := app.Dial(&routing.Addr{PubKey: rpk, Port: 3})
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.Equal(t, rpk.Hex()+":3", conn.RemoteAddr().String())
	assert.Equal(t, lpk.Hex()+":2", conn.LocalAddr().String())

	require.NotNil(t, app.conns[LoopAddr{Port: 2, Remote: routing.Addr{PubKey: rpk, Port: 3}}])
	require.NoError(t, conn.Close())

	// Justified. Attempt to remove produces: FAIL
	time.Sleep(100 * time.Millisecond)

	addr := &LoopAddr{}
	require.NoError(t, json.Unmarshal(<-dataCh, addr))
	assert.Equal(t, uint16(2), addr.Port)
	assert.Equal(t, rpk, addr.Remote.PubKey)
	assert.Equal(t, uint16(3), addr.Remote.Port)

	app.mu.Lock()
	require.Len(t, app.conns, 0)
	app.mu.Unlock()
	require.NoError(t, proto.Close())
}

func TestAppAccept(t *testing.T) {
	lpk, _ := cipher.GenerateKeyPair()
	rpk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	app := &App{proto: NewProtocol(in), acceptChan: make(chan [2]*routing.Addr), conns: make(map[LoopAddr]io.ReadWriteCloser)}
	go app.handleProto()

	proto := NewProtocol(out)
	go proto.Serve(nil) // nolint: errcheck

	connCh := make(chan net.Conn)
	errCh := make(chan error)
	go func() {
		conn, err := app.Accept()
		errCh <- err
		connCh <- conn
	}()

	require.NoError(t, proto.Send(FrameConfirmLoop, [2]*routing.Addr{{PubKey: lpk, Port: 2}, {PubKey: rpk, Port: 3}}, nil))

	require.NoError(t, <-errCh)
	conn := <-connCh
	require.NotNil(t, conn)
	assert.Equal(t, rpk.Hex()+":3", conn.RemoteAddr().String())
	assert.Equal(t, lpk.Hex()+":2", conn.LocalAddr().String())
	require.Len(t, app.conns, 1)

	go func() {
		conn, err := app.Accept()
		errCh <- err
		connCh <- conn
	}()

	require.NoError(t, proto.Send(FrameConfirmLoop, [2]*routing.Addr{{PubKey: lpk, Port: 2}, {PubKey: rpk, Port: 2}}, nil))

	require.NoError(t, <-errCh)
	conn = <-connCh
	require.NotNil(t, conn)
	assert.Equal(t, rpk.Hex()+":2", conn.RemoteAddr().String())
	assert.Equal(t, lpk.Hex()+":2", conn.LocalAddr().String())
	require.Len(t, app.conns, 2)
}

func TestAppWrite(t *testing.T) {
	rpk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	appIn, appOut := net.Pipe()
	app := &App{proto: NewProtocol(in)}
	go app.handleProto()
	go app.serveConn(&LoopAddr{Port: 2, Remote: routing.Addr{PubKey: rpk, Port: 3}}, appIn)

	proto := NewProtocol(out)
	dataCh := make(chan []byte)
	go func() {
		proto.Serve(func(f Frame, p []byte) (interface{}, error) { // nolint: errcheck
			if f != FrameSend {
				return nil, errors.New("unexpected frame")
			}

			go func() { dataCh <- p }()
			return nil, nil
		})
	}()

	n, err := appOut.Write([]byte("foo"))
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	packet := &Packet{}
	require.NoError(t, json.Unmarshal(<-dataCh, packet))
	assert.Equal(t, rpk, packet.Addr.Remote.PubKey)
	assert.Equal(t, uint16(3), packet.Addr.Remote.Port)
	assert.Equal(t, uint16(2), packet.Addr.Port)
	assert.Equal(t, []byte("foo"), packet.Payload)

	require.NoError(t, proto.Close())
	require.NoError(t, appOut.Close())
}

func TestAppRead(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	appIn, appOut := net.Pipe()
	app := &App{proto: NewProtocol(in), conns: map[LoopAddr]io.ReadWriteCloser{LoopAddr{Port: 2, Remote: routing.Addr{PubKey: pk, Port: 3}}: appIn}}
	go app.handleProto()

	proto := NewProtocol(out)
	go proto.Serve(nil) // nolint: errcheck

	errCh := make(chan error)
	go func() {
		errCh <- proto.Send(FrameSend, &Packet{&LoopAddr{Port: 2, Remote: routing.Addr{PubKey: pk, Port: 3}}, []byte("foo")}, nil)
	}()

	buf := make([]byte, 3)
	n, err := appOut.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)

	require.NoError(t, <-errCh)

	require.NoError(t, proto.Close())
	require.NoError(t, appOut.Close())
}

func TestAppSetup(t *testing.T) {
	srvConn, clientConn, err := OpenPipeConn()
	require.NoError(t, err)

	srvConn.SetDeadline(time.Now().Add(time.Second))    // nolint: errcheck
	clientConn.SetDeadline(time.Now().Add(time.Second)) // nolint: errcheck

	proto := NewProtocol(srvConn)
	dataCh := make(chan []byte)
	go proto.Serve(func(f Frame, p []byte) (interface{}, error) { // nolint: errcheck, unparam
		if f != FrameInit {
			return nil, errors.New("unexpected frame")
		}

		go func() { dataCh <- p }()
		return nil, nil
	})

	inFd, outFd := clientConn.Fd()
	_, err = SetupFromPipe(&Config{AppName: "foo", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, inFd, outFd)
	require.NoError(t, err)

	config := &Config{}
	require.NoError(t, json.Unmarshal(<-dataCh, config))
	assert.Equal(t, "foo", config.AppName)
	assert.Equal(t, "0.0.1", config.AppVersion)
	assert.Equal(t, "0.0.1", config.ProtocolVersion)

	require.NoError(t, proto.Close())
}

func TestAppCloseConn(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	appIn, appOut := net.Pipe()
	app := &App{proto: NewProtocol(in), conns: map[LoopAddr]io.ReadWriteCloser{LoopAddr{Port: 2, Remote: routing.Addr{PubKey: pk, Port: 3}}: appIn}}
	go app.handleProto()

	proto := NewProtocol(out)
	go proto.Serve(nil) // nolint: errcheck

	errCh := make(chan error)
	go func() {
		errCh <- proto.Send(FrameClose, &LoopAddr{Port: 2, Remote: routing.Addr{PubKey: pk, Port: 3}}, nil)
	}()

	_, err := appOut.Read(make([]byte, 3))
	require.Equal(t, io.EOF, err)
	require.Len(t, app.conns, 0)
}

func TestAppClose(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	appIn, appOut := net.Pipe()
	app := &App{proto: NewProtocol(in), conns: map[LoopAddr]io.ReadWriteCloser{LoopAddr{Port: 2, Remote: routing.Addr{PubKey: pk, Port: 3}}: appIn}, doneChan: make(chan struct{})}
	go app.handleProto()

	proto := NewProtocol(out)
	dataCh := make(chan []byte)
	go proto.Serve(func(f Frame, p []byte) (interface{}, error) { // nolint: errcheck, unparam
		if f != FrameClose {
			return nil, errors.New("unexpected frame")
		}

		go func() { dataCh <- p }()
		return nil, nil
	})

	require.NoError(t, app.Close())

	_, err := appOut.Read(make([]byte, 3))
	require.Equal(t, io.EOF, err)

	addr := &LoopAddr{}
	require.NoError(t, json.Unmarshal(<-dataCh, addr))
	assert.Equal(t, uint16(2), addr.Port)
	assert.Equal(t, pk, addr.Remote.PubKey)
	assert.Equal(t, uint16(3), addr.Remote.Port)
}

func TestAppCommand(t *testing.T) {
	conn, cmd, err := Command(&Config{}, "/apps", nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, cmd)
}
