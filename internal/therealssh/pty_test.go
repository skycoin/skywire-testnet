package therealssh

import (
	"fmt"
	"github.com/creack/pty"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"os/exec"
	"os/user"
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

func TestRunInPTY(t *testing.T) {
	dialConn, acceptConn := net.Pipe()
	pd := PipeDialer{PipeWithRoutingAddr{dialConn}, acceptConn}
	_, client, err := NewClient(":9999", pd)
	require.NoError(t,err)

	server := NewServer(MockAuthorizer{})
	go func() {
		require.NoError(t, server.Serve(PipeWithRoutingAddr{acceptConn}))
	}()

	_, ch, err := client.OpenChannel(cipher.PubKey{})
	require.NoError(t, err)

	cuser, err := user.Current()
	require.NoError(t, err)

	args := RequestPTYArgs{
		Username: cuser.Username,
		RemotePK: cipher.PubKey{},
		Size:     &pty.Winsize{
			Rows: 100,
			Cols: 100,
			X:    100,
			Y:    100,
		},
	}
	res, err := ch.Request(RequestPTY, args.ToBinary())
	require.NoError(t, err)
	fmt.Println(res)

	res, err = ch.Request(RequestExec, []byte("ls"))
	require.NoError(t, err)
	fmt.Println(res)
}

type MockAuthorizer struct {}

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