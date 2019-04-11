package app

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/pkg/cipher"
)

func TestDial(t *testing.T) {

}

func TestAppDial(t *testing.T) {
	lpk, _ := cipher.GenerateKeyPair()
	rpk, _ := cipher.GenerateKeyPair()

	in, out := net.Pipe()
	proto := appnet.NewProtocol(out)
	app := &App{proto: appnet.NewProtocol(in), connMap: make(map[LoopMeta]io.ReadWriteCloser)}
	go app.handleProto()

	dataCh := make(chan []byte)
	go proto.Serve(func(f appnet.FrameType, p []byte) (interface{}, error) { // nolint: errcheck
		if f == appnet.FrameCreateLoop {
			return &appnet.LoopAddr{PubKey: lpk, Port: 2}, nil
		}

		if f == appnet.FrameCloseLoop {
			go func() { dataCh <- p }()
			return nil, nil
		}

		return nil, errors.New("unexpected frame")
	})
	conn, err := app.Dial(&appnet.LoopAddr{PubKey: rpk, Port: 3})
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.Equal(t, rpk.Hex()+":3", conn.RemoteAddr().String())
	assert.Equal(t, lpk.Hex()+":2", conn.LocalAddr().String())

	require.NotNil(t, app.connMap[LoopMeta{2, appnet.LoopAddr{rpk, 3}}])
	require.NoError(t, conn.Close())

	time.Sleep(100 * time.Millisecond)

	addr := &LoopMeta{}
	require.NoError(t, json.Unmarshal(<-dataCh, addr))
	assert.Equal(t, uint16(2), addr.LocalPort)
	assert.Equal(t, rpk, addr.Remote.PubKey)
	assert.Equal(t, uint16(3), addr.Remote.Port)

	app.mu.Lock()
	require.Len(t, app.connMap, 0)
	app.mu.Unlock()
	require.NoError(t, proto.Close())
}

func TestAppAccept(t *testing.T) {
	lpk, _ := cipher.GenerateKeyPair()
	rpk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	app := &App{proto: appnet.NewProtocol(in), acceptChan: make(chan [2]*appnet.LoopAddr), connMap: make(map[LoopMeta]io.ReadWriteCloser)}
	go app.handleProto()

	proto := appnet.NewProtocol(out)
	go proto.Serve(nil) // nolint: errcheck

	connCh := make(chan net.Conn)
	errCh := make(chan error)
	go func() {
		conn, err := app.Accept()
		errCh <- err
		connCh <- conn
	}()

	require.NoError(t, proto.Send(appnet.FrameConfirmLoop, [2]*appnet.LoopAddr{&appnet.LoopAddr{lpk, 2}, &appnet.LoopAddr{rpk, 3}}, nil))

	require.NoError(t, <-errCh)
	conn := <-connCh
	require.NotNil(t, conn)
	assert.Equal(t, rpk.Hex()+":3", conn.RemoteAddr().String())
	assert.Equal(t, lpk.Hex()+":2", conn.LocalAddr().String())
	require.Len(t, app.connMap, 1)

	go func() {
		conn, err := app.Accept()
		errCh <- err
		connCh <- conn
	}()

	require.NoError(t, proto.Send(appnet.FrameConfirmLoop, [2]*appnet.LoopAddr{&appnet.LoopAddr{lpk, 2}, &appnet.LoopAddr{rpk, 2}}, nil))

	require.NoError(t, <-errCh)
	conn = <-connCh
	require.NotNil(t, conn)
	assert.Equal(t, rpk.Hex()+":2", conn.RemoteAddr().String())
	assert.Equal(t, lpk.Hex()+":2", conn.LocalAddr().String())
	require.Len(t, app.connMap, 2)
}

func TestAppWrite(t *testing.T) {
	rpk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	appIn, appOut := net.Pipe()
	app := &App{proto: appnet.NewProtocol(in)}
	go app.handleProto()
	go app.serveConn(&LoopMeta{2, appnet.LoopAddr{rpk, 3}}, appIn)

	proto := appnet.NewProtocol(out)
	dataCh := make(chan []byte)
	go func() {
		proto.Serve(func(f appnet.FrameType, p []byte) (interface{}, error) { // nolint: errcheck
			if f != appnet.FrameData {
				return nil, errors.New("unexpected frame")
			}

			go func() { dataCh <- p }()
			return nil, nil
		})
	}()

	n, err := appOut.Write([]byte("foo"))
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	packet := &DataFrame{}
	require.NoError(t, json.Unmarshal(<-dataCh, packet))
	assert.Equal(t, rpk, packet.Meta.Remote.PubKey)
	assert.Equal(t, uint16(3), packet.Meta.Remote.Port)
	assert.Equal(t, uint16(2), packet.Meta.LocalPort)
	assert.Equal(t, []byte("foo"), packet.Data)

	require.NoError(t, proto.Close())
	require.NoError(t, appOut.Close())
}

func TestAppRead(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	appIn, appOut := net.Pipe()
	app := &App{proto: appnet.NewProtocol(in), connMap: map[LoopMeta]io.ReadWriteCloser{LoopMeta{2, appnet.LoopAddr{pk, 3}}: appIn}}
	go app.handleProto()

	proto := appnet.NewProtocol(out)
	go proto.Serve(nil) // nolint: errcheck

	errCh := make(chan error)
	go func() {
		errCh <- proto.Send(appnet.FrameData, &DataFrame{&LoopMeta{2, appnet.LoopAddr{pk, 3}}, []byte("foo")}, nil)
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
	srvConn, clientConn, err := appnet.OpenPipeConn()
	require.NoError(t, err)

	srvConn.SetDeadline(time.Now().Add(time.Second))    // nolint: errcheck
	clientConn.SetDeadline(time.Now().Add(time.Second)) // nolint: errcheck

	proto := appnet.NewProtocol(srvConn)
	dataCh := make(chan []byte)
	go proto.Serve(func(f appnet.FrameType, p []byte) (interface{}, error) { // nolint: errcheck, unparam
		if f != appnet.FrameInit {
			return nil, errors.New("unexpected frame")
		}

		go func() { dataCh <- p }()
		return nil, nil
	})

	inFd, outFd := clientConn.Fd()
	_, err = SetupFromPipe(&Config{AppName: "foo", AppVersion: "0.0.1", protocolVersion: "0.0.1"}, inFd, outFd)
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
	app := &App{proto: appnet.NewProtocol(in), connMap: map[LoopMeta]io.ReadWriteCloser{LoopMeta{2, appnet.LoopAddr{pk, 3}}: appIn}}
	go app.handleProto()

	proto := appnet.NewProtocol(out)
	go proto.Serve(nil) // nolint: errcheck

	errCh := make(chan error)
	go func() {
		errCh <- proto.Send(appnet.FrameCloseLoop, &LoopMeta{2, appnet.LoopAddr{pk, 3}}, nil)
	}()

	_, err := appOut.Read(make([]byte, 3))
	require.Equal(t, io.EOF, err)
	require.Len(t, app.connMap, 0)
}

func TestAppClose(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	appIn, appOut := net.Pipe()
	app := &App{proto: appnet.NewProtocol(in), connMap: map[LoopMeta]io.ReadWriteCloser{LoopMeta{2, appnet.LoopAddr{pk, 3}}: appIn}, doneChan: make(chan struct{})}
	go app.handleProto()

	proto := appnet.NewProtocol(out)
	dataCh := make(chan []byte)
	go proto.Serve(func(f appnet.FrameType, p []byte) (interface{}, error) { // nolint: errcheck, unparam
		if f != appnet.FrameCloseLoop {
			return nil, errors.New("unexpected frame")
		}

		go func() { dataCh <- p }()
		return nil, nil
	})

	require.NoError(t, app.Close())

	_, err := appOut.Read(make([]byte, 3))
	require.Equal(t, io.EOF, err)

	addr := &LoopMeta{}
	require.NoError(t, json.Unmarshal(<-dataCh, addr))
	assert.Equal(t, uint16(2), addr.LocalPort)
	assert.Equal(t, pk, addr.Remote.PubKey)
	assert.Equal(t, uint16(3), addr.Remote.Port)
}

func TestAppCommand(t *testing.T) {
	conn, cmd, err := Command(&Config{}, "/apps", nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, cmd)
}
