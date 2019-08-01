package therealssh

import (
	"errors"
	"fmt"
	"github.com/creack/pty"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"os/exec"
	"testing"
)

func runInPTY(command string) ([]byte, error) {
	c := exec.Command(command)
	ptmx, err := pty.Start(c)
	if err != nil {
		return nil, err
	}

	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// as stated in https://github.com/creack/pty/issues/21#issuecomment-513069505 we can ignore this error
	res, _ := ioutil.ReadAll(ptmx) // nolint: err
	return res, nil
}

func runInRemote(t *testing.T, pk cipher.PubKey) ([]byte, error) {
	in, out := net.Pipe()
	ch := OpenChannel(1, routing.Addr{PubKey: pk, Port: Port}, in)

	errCh := make(chan error)
	go func() {
		errCh <- ch.Send(CmdChannelOpen, []byte("foo"))
	}()

	server := NewServer(MockAuthorizer{})
	go func(){
		err := server.Serve(out)
		require.NoError(t, err)
	}()

	type data struct {
		res []byte
		err error
	}
	resCh := make(chan data)
	go func() {
		res, err := ch.Request(RequestExec, []byte("ls"))
		resCh <- data{res, err}
	}()

	d := <- resCh
	require.NoError(t, d.err)
	fmt.Println(d.res)
	return d.res, d.err
}

func TestRunInPTY(t *testing.T) {
	serverPK, _ := cipher.GenerateKeyPair()
	//clientPK, _ := cipher.GenerateKeyPair()

	clientConn, serverConn := net.Pipe()

	serverApp := createDefaultServerApp(t, serverConn)
	clientApp := createDefaultClientApp(t, clientConn)


	// server.Serve
	server := NewServer(MockAuthorizer{})
	go func() {
		conn, err := serverApp.Accept()
		require.NoError(t, err)

		require.NoError(t, server.Serve(conn))
	}()

	// client
	_, client, err := NewClient(":9999", clientApp)
	require.NoError(t, err)

	go func() {
		conn, err := clientApp.Dial(routing.Addr{PubKey: serverPK, Port: 2})
		require.NoError(t, err)

		require.NoError(t, client.serveConn(conn))
	}()

	_, ch, err := client.OpenChannel(serverPK)
	require.NoError(t, err)
	res, err := ch.Request(RequestExec, []byte("ls"))
	require.NoError(t, err)
	fmt.Println(res)

	require.NoError(t, serverApp.Close())
	require.NoError(t, clientApp.Close())
	require.NoError(t, server.Close())
	require.NoError(t, client.Close())
}

func createDefaultServerApp(t *testing.T, conn net.Conn) *app.App {
	sshApp := app.NewAppMock(conn)

	go func() {
		f := func(f app.Frame, p []byte) (interface{}, error) {
			if f == app.FrameCreateLoop {
				return &routing.Addr{PubKey: lpk, Port: 2}, nil
			}

			if f == app.FrameClose {
				go func() { dataCh <- p }()
				return nil, nil
			}

			return nil, errors.New("unexpected frame")
		}
	}()

	return sshApp
}

func createDefaultClientApp(t *testing.T, conn net.Conn) *app.App {
	sshApp := app.NewAppMock(conn)
	go func() {
		f := func(f app.Frame, p []byte) (interface{}, error) {
			if f == app.FrameCreateLoop {
				return &routing.Addr{PubKey: lpk, Port: 2}, nil
			}

			if f == app.FrameClose {
				return nil, nil
			}

			return nil, errors.New("unexpected frame")
		}
		serveErrCh <- proto.Serve(f)
	}()

	return sshApp
}

type MockAuthorizer struct {}

func (MockAuthorizer) Authorize(pk cipher.PubKey) error {
	return nil
}
