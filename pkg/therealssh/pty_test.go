// +build dragonfly freebsd linux netbsd openbsd

package therealssh

import (
	"net"
	"net/http"
	"net/rpc"
	"os/user"
	"testing"

	"github.com/creack/pty"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/skycoin/src/util/logging"
	"github.com/stretchr/testify/require"

	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
)

func TestRunRPC(t *testing.T) {
	dialConn, acceptConn := net.Pipe()
	pd := PipeDialer{PipeWithRoutingAddr{dialConn}, acceptConn}
	rpcC, client, err := NewClient(":9998", pd)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, client.Close())
	}()

	server := NewServer(MockAuthorizer{}, logging.NewMasterLogger())
	go func() {
		server.Serve(PipeWithRoutingAddr{acceptConn}) // nolint
	}()

	go func() {
		http.Serve(rpcC, nil) // nolint
	}()

	rpcD, err := rpc.DialHTTP("tcp", ":9998")
	require.NoError(t, err)

	cuser, err := user.Current()
	require.NoError(t, err)

	ptyArgs := &RequestPTYArgs{
		Username: cuser.Username,
		RemotePK: cipher.PubKey{},
		Size: &pty.Winsize{
			Rows: 100,
			Cols: 100,
			X:    100,
			Y:    100,
		},
	}
	var channel uint32
	err = rpcD.Call("RPCClient.RequestPTY", ptyArgs, &channel)
	require.NoError(t, err)

	var socketPath string
	execArgs := &ExecArgs{
		ChannelID:       channel,
		CommandWithArgs: []string{"ls"},
	}

	err = rpcD.Call("RPCClient.Run", execArgs, &socketPath)
	require.NoError(t, err)

	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
	require.NoError(t, err)

	b := make([]byte, 6024)
	_, err = conn.Read(b)
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
