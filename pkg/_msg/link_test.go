package messaging

import (
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

var (
	initPK, initSK = genKeyPair("initiator seed")
	respPK, respSK = genKeyPair("responder seed")
)

func genKeyPair(seed string) (cipher.PubKey, cipher.SecKey) {
	pk, sk, err := cipher.GenerateDeterministicKeyPair([]byte(seed))
	if err != nil {
		panic(err)
	}
	return pk, sk
}

type callbacksModifier func(cb *Callbacks)

type mockReadWriteCloser struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (rw mockReadWriteCloser) Read(p []byte) (n int, err error) {
	return rw.reader.Read(p)
}

func (rw mockReadWriteCloser) Write(p []byte) (n int, err error) {
	return rw.writer.Write(p)
}

func (rw mockReadWriteCloser) Close() error {
	if err := rw.writer.Close(); err != nil {
		return err
	}
	if err := rw.reader.Close(); err != nil {
		return err
	}
	return nil
}

func makeMockPipe() (a, b mockReadWriteCloser) {
	a.reader, b.writer = io.Pipe()
	b.reader, a.writer = io.Pipe()
	return
}

func makeConnPair(t *testing.T,
	initPK, respPK cipher.PubKey,
	initSK, respSK cipher.SecKey,
	modifyInit, modifyResp callbacksModifier,
) (initConn *Link, respConn *Link) {

	initNet, respNet := makeMockPipe()

	initConfig := DefaultLinkConfig()
	initConfig.Public = initPK
	initConfig.Secret = initSK
	initConfig.Remote = respPK
	initConfig.Initiator = true

	respConfig := DefaultLinkConfig()
	respConfig.Public = respPK
	respConfig.Secret = respSK

	initCB := connCallbacks()
	modifyInit(initCB)

	respCB := connCallbacks()
	modifyResp(respCB)

	initConn, err := NewLink(&initNet, initConfig, initCB)
	require.NoError(t, err)

	respConn, err = NewLink(&respNet, respConfig, respCB)
	require.NoError(t, err)

	return initConn, respConn
}

func TestNewConn(t *testing.T) {
	// An initiator and responder can send messages to one another.
	t.Run("send_messages_back_and_forth", func(t *testing.T) {
		const expectedMsgCount = 10000

		msgsWG := new(sync.WaitGroup)    // records the send and receive event of each msg.
		msgsWG.Add(expectedMsgCount * 2) // 10000 msgs x (send event + receive event)

		initConn, respConn := makeConnPair(t,
			initPK, respPK,
			initSK, respSK,
			func(cb *Callbacks) { // Initiator callbacks.
				cb.Data = func(conn *Link, dt FrameType, body []byte) error {
					msgsWG.Done()
					return nil
				}
			},
			func(cb *Callbacks) { // Responder callbacks.
				cb.Data = func(conn *Link, dt FrameType, body []byte) error {
					msgsWG.Done()
					_, err := conn.Send(1, []byte(fmt.Sprintf("got msg: %s", string(body))))
					return err
				}
			},
		)

		initConnOpenDone := make(chan error)
		connsWG := new(sync.WaitGroup)
		go func() { initConnOpenDone <- initConn.Open(connsWG) }()
		require.NoError(t, respConn.Open(connsWG))
		require.NoError(t, <-initConnOpenDone)

		for i := 0; i < expectedMsgCount; i++ {
			msg := fmt.Sprintf("Hello world %d!", i)
			_, err := initConn.Send(1, []byte(msg))
			require.NoError(t, err)
		}

		msgsWG.Wait()

		require.NoError(t, initConn.Close())
		connsWG.Wait()
	})
}

func connHandshakeCompleteAction() HandshakeCompleteAction {
	return func(conn *Link) {}
}

func connMessageAction() FrameAction {
	return func(conn *Link, dt FrameType, body []byte) error {
		conn.logf("%s", string(body))
		return nil
	}
}

func connCloseAction() TCPCloseAction {
	return func(conn *Link, remote bool) {
		conn.logf("connection closed: Remote(%v)", remote)
	}
}

func connCallbacks() *Callbacks {
	return &Callbacks{
		HandshakeComplete: connHandshakeCompleteAction(),
		Data:              connMessageAction(),
		Close:             connCloseAction(),
	}
}
