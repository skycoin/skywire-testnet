package app

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/pkg/cipher"
)

func randLoopAddr() *LoopAddr {
	pk, _ := cipher.GenerateKeyPair()
	port := binary.BigEndian.Uint16(cipher.RandByte(2))
	return &LoopAddr{PubKey: pk, Port: port}
}

func randLoopMeta() LoopMeta {
	return LoopMeta{
		Local:  *randLoopAddr(),
		Remote: *randLoopAddr(),
	}
}

func setupApp(t *testing.T, appConn net.Conn, hostPK cipher.PubKey) func() {
	_meta = Meta{
		AppName:         "test-app",
		AppVersion:      "v0.0.0",
		ProtocolVersion: protocolVersion,
		Host:            hostPK,
	}

	_mu.Lock()
	_proto = appnet.NewProtocol(appConn)
	_acceptCh = make(chan LoopMeta)
	_loopPipes = make(map[LoopMeta]io.ReadWriteCloser)
	_mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveHostConn()
	}()

	return func() {
		require.NoError(t, Close())
		require.NoError(t, <-errCh)
	}
}

func setupHost(t *testing.T, hostConn net.Conn, handlers appnet.HandlerMap) (*appnet.Protocol, func()) {
	proto := appnet.NewProtocol(hostConn)

	errCh := make(chan error, 1)
	go func() { errCh <- proto.Serve(handlers) }()

	return proto, func() {
		require.NoError(t, proto.Close())
		require.NoError(t, <-errCh)
	}
}

func setup(t *testing.T, hostPK cipher.PubKey, hostHandlers appnet.HandlerMap) (*appnet.Protocol, func()) {
	appConn, hostConn := net.Pipe()
	var (
		closeApp          = setupApp(t, appConn, hostPK)
		hProto, closeHost = setupHost(t, hostConn, hostHandlers)
	)
	return hProto, func() {
		closeHost()
		closeApp()
	}
}

func TestDial(t *testing.T) {
	lm := randLoopMeta()

	const frameCount = 10

	hCloseCh := make(chan []byte, 1)
	hFrameCh := make(chan []byte, frameCount)

	_, teardown := setup(t, lm.Local.PubKey, appnet.HandlerMap{
		appnet.FrameCreateLoop: func(p *appnet.Protocol, b []byte) ([]byte, error) {
			return lm.Encode(), nil
		},
		appnet.FrameCloseLoop: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			hCloseCh <- b
			return nil, nil
		},
		appnet.FrameData: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			hFrameCh <- b
			return nil, nil
		},
	})

	defer teardown()

	// Check dial.
	loop, err := Dial(lm.Remote)
	require.NoError(t, err)
	require.NotNil(t, loop)
	assert.Equal(t, &lm.Local, loop.LocalAddr())
	assert.Equal(t, &lm.Remote, loop.RemoteAddr())

	// Write/read to/from loop (between host node and app).
	for i := 0; i < frameCount; i++ {
		payload := []byte(fmt.Sprintf("Hello world %d! Random Data: %v", i, cipher.RandByte(10)))
		n, err := loop.Write(payload)
		assert.NoError(t, err)
		assert.Equal(t, len(payload), n)

		var df DataFrame
		require.NoError(t, df.Decode(<-hFrameCh))
		assert.Equal(t, lm, df.Meta)
		assert.Equal(t, payload, df.Data)
	}

	// Ensure loop pipes are successfully created.
	_mu.RLock()
	_, ok := _loopPipes[lm]
	_mu.RUnlock()
	require.True(t, ok)

	require.NoError(t, loop.Close())

	// Check response from host node is correct.
	var obtainedLM LoopMeta
	require.NoError(t, obtainedLM.Decode(<-hCloseCh))
	assert.Equal(t, lm, obtainedLM)

	_mu.Lock()
	assert.Len(t, _loopPipes, 0)
	_mu.Unlock()
}

func TestAccept(t *testing.T) {
	lAddr := randLoopAddr()

	hProto, teardown := setup(t, lAddr.PubKey, nil)
	defer teardown()

	const count = 10

	type Result struct {
		Loop net.Conn
		Err  error
	}
	resultCh := make(chan Result)
	defer close(resultCh)

	for i := 0; i < count; i++ {
		go func() {
			loop, err := Accept()
			resultCh <- Result{Loop: loop, Err: err}
		}()

		lm := LoopMeta{Local: *lAddr, Remote: *randLoopAddr()}
		_, err := hProto.Call(appnet.FrameConfirmLoop, lm.Encode())
		require.NoError(t, err)

		r := <-resultCh
		require.NoError(t, r.Err)
		require.NotNil(t, r.Loop)
		assert.Equal(t, &lm.Local, r.Loop.LocalAddr())
		assert.Equal(t, &lm.Remote, r.Loop.RemoteAddr())

		_mu.RLock()
		assert.Len(t, _loopPipes, i+1)
		_mu.RUnlock()

		fmt.Println("RUN", i, "(OK)")
	}
}

func TestLoopConn_Write(t *testing.T) {
	lm := randLoopMeta()

	const dataCount = 10
	dataCh := make(chan []byte, dataCount)
	defer close(dataCh)

	_, teardown := setup(t, lm.Local.PubKey, appnet.HandlerMap{
		appnet.FrameData: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			dataCh <- b
			return nil, nil
		},
	})

	defer teardown()

	for i := 0; i < dataCount; i++ {
		payload := fmt.Sprintf("This is payload number %d! Random data: %v", i, cipher.RandByte(10))
		_, err := _proto.Call(appnet.FrameData, []byte(payload))
		require.NoError(t, err)

		require.Equal(t, payload, string(<-dataCh))
	}
}
