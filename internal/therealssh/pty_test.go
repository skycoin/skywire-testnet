package therealssh

import (
	"fmt"
	"net"
	"os/user"
	"testing"

	"github.com/creack/pty"
	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/routing"
)

func TestRunInPTY(t *testing.T) {
	dialConn, acceptConn := net.Pipe()
	pd := PipeDialer{PipeWithRoutingAddr{dialConn}, acceptConn}
	_, client, err := NewClient(":9999", pd)
	require.NoError(t, err)

	server := NewServer(MockAuthorizer{})
	go func() {
		err := server.Serve(PipeWithRoutingAddr{acceptConn})
		fmt.Println("server.Serve finished with err: ", err)
		require.NoError(t, err)
	}()

	_, ch, err := client.OpenChannel(cipher.PubKey{})
	require.NoError(t, err)

	cuser, err := user.Current()
	require.NoError(t, err)

	args := RequestPTYArgs{
		Username: cuser.Username,
		RemotePK: cipher.PubKey{},
		Size: &pty.Winsize{
			Rows: 100,
			Cols: 100,
			X:    100,
			Y:    100,
		},
	}
	_, err = ch.Request(RequestPTY, args.ToBinary())
	require.NoError(t, err)

	_, err = ch.Request(RequestExecWithoutShell, []byte("ls"))
	require.NoError(t, err)

	b := make([]byte, 6024)
	_, err = ch.Read(b)
	require.NoError(t, err)
	require.Contains(t, string(b), "pty_test.go")
}

type MockAuthorizer struct{}

func (MockAuthorizer) Authorize(pk cipher.PubKey) error {
	return nil
}

type PipeDialer struct {
	dialConn, acceptConn net.Conn
}

func (p PipeDialer) Dial(raddr routing.Addr) (c net.Conn, err error) {
	return p.dialConn, nil
}

type PipeWithRoutingAddr struct {
	net.Conn
}

func (p PipeWithRoutingAddr) RemoteAddr() net.Addr {
	return routing.Addr{
		PubKey: cipher.PubKey{},
		Port:   9999,
	}
}
