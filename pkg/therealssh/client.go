package therealssh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/internal/netutil"
	"github.com/skycoin/skywire/pkg/routing"
)

var r = netutil.NewRetrier(50*time.Millisecond, 5, 2)

// Client proxies CLI's requests to a remote server. Control messages
// are sent via RPC interface. PTY data is exchanged via unix socket.
type Client struct {
	dialer dialer
	chans  *chanList
}

// NewClient construct new RPC listener and Client from a given RPC address and app dialer.
func NewClient(rpcAddr string, d dialer) (net.Listener, *Client, error) {
	client := &Client{chans: newChanList(), dialer: d}
	rpcClient := &RPCClient{client}
	if err := rpc.Register(rpcClient); err != nil {
		return nil, nil, fmt.Errorf("RPC register failure: %s", err)
	}
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("RPC listen failure: %s", err)
	}

	return l, client, nil
}

// OpenChannel requests new Channel on the remote Server.
func (c *Client) OpenChannel(remotePK cipher.PubKey) (localID uint32, sshCh *SSHChannel, cErr error) {
	var conn net.Conn
	var err error

	err = r.Do(func() error {
		conn, err = c.dialer.Dial(routing.Addr{PubKey: remotePK, Port: Port})
		return err
	})
	if err != nil {
		cErr = fmt.Errorf("dial failed: %s", err)
		return
	}

	sshCh = OpenClientChannel(0, remotePK, conn)
	Log.Debugln("sending channel open command")
	localID = c.chans.add(sshCh)
	req := appendU32([]byte{byte(CmdChannelOpen)}, localID)
	if _, err := conn.Write(req); err != nil {
		cErr = fmt.Errorf("failed to send open channel request: %s", err)
		return
	}

	go func() {
		if err := c.serveConn(conn); err != nil {
			Log.Error(err)
		}
	}()

	Log.Debugln("waiting for channel open response")
	data := <-sshCh.msgCh
	Log.Debugln("got channel open response")
	if data[0] == ResponseFail {
		cErr = fmt.Errorf("failed to open channel: %s", string(data[1:]))
		return
	}

	sshCh.RemoteID = binary.BigEndian.Uint32(data[1:])
	return localID, sshCh, cErr
}

func (c *Client) resolveChannel(remotePK cipher.PubKey, localID uint32) (*SSHChannel, error) {
	sshCh := c.chans.getChannel(localID)
	if sshCh == nil {
		return nil, errors.New("channel is not opened")
	}

	if sshCh.RemoteAddr.PubKey != remotePK {
		return nil, errors.New("unauthorized")
	}

	return sshCh, nil
}

// Route defines routing rules for received App messages.
func (c *Client) serveConn(conn net.Conn) error {
	for {
		buf := make([]byte, 32*1024)
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		raddr := conn.RemoteAddr().(routing.Addr)
		payload := buf[:n]

		if len(payload) < 5 {
			return errors.New("corrupted payload")
		}

		sshCh, err := c.resolveChannel(raddr.PubKey, binary.BigEndian.Uint32(payload[1:]))
		if err != nil {
			return err
		}

		data := payload[5:]
		Log.Debugf("got new command: %x", payload[0])
		switch CommandType(payload[0]) {
		case CmdChannelOpenResponse, CmdChannelResponse:
			sshCh.msgCh <- data
		case CmdChannelData:
			sshCh.dataChMx.Lock()
			if !sshCh.IsClosed() {
				sshCh.dataCh <- data
			}
			sshCh.dataChMx.Unlock()
		case CmdChannelServerClose:
			err = sshCh.Close()
		default:
			err = fmt.Errorf("unknown command: %x", payload[0])
		}

		if err != nil {
			return err
		}
	}
}

// Close closes all opened channels.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	for _, sshCh := range c.chans.dropAll() {
		if err := sshCh.Close(); err != nil {
			Log.WithError(err).Warn("Failed to close SSH channel")
		}
	}

	return nil
}

// RPCClient exposes Client's methods via RPC interface.
type RPCClient struct {
	c *Client
}

// RequestPTY defines RPC request for a new PTY session.
func (rpc *RPCClient) RequestPTY(args *RequestPTYArgs, channelID *uint32) error {
	Log.Debugln("requesting SSH channel")
	localID, channel, err := rpc.c.OpenChannel(args.RemotePK)
	if err != nil {
		return err
	}

	Log.Debugln("requesting PTY session")
	if _, err := channel.Request(RequestPTY, args.ToBinary()); err != nil {
		return fmt.Errorf("PTY request failure: %s", err)
	}

	*channelID = localID
	return nil
}

// Exec defines new remote execution RPC request.
func (rpc *RPCClient) Exec(args *ExecArgs, socketPath *string) error {
	sshCh := rpc.c.chans.getChannel(args.ChannelID)
	if sshCh == nil {
		return errors.New("unknown channel")
	}

	Log.Debugln("requesting shell process")
	if args.CommandWithArgs == nil {
		if _, err := sshCh.Request(RequestShell, nil); err != nil {
			return fmt.Errorf("shell request failure: %s", err)
		}
	} else {
		if _, err := sshCh.Request(RequestExec, args.ToBinary()); err != nil {
			return fmt.Errorf("shell request failure: %s", err)
		}
	}

	waitCh := make(chan bool)
	go func() {
		Log.Debugln("starting socket listener")
		waitCh <- true
		if err := sshCh.ServeSocket(); err != nil {
			Log.Error("Session failure:", err)
		}
	}()

	*socketPath = sshCh.SocketPath()
	<-waitCh
	return nil
}

// Run defines new remote shell-less execution RPC request.
func (rpc *RPCClient) Run(args *ExecArgs, socketPath *string) error {
	sshCh := rpc.c.chans.getChannel(args.ChannelID)
	if sshCh == nil {
		return errors.New("unknown channel")
	}

	Log.Debugln("requesting shell-less process execution")
	if _, err := sshCh.Request(RequestExecWithoutShell, args.ToBinary()); err != nil {
		return fmt.Errorf("run command request failure: %s", err)
	}

	waitCh := make(chan bool)
	go func() {
		Log.Debugln("starting socket listener")
		waitCh <- true
		if err := sshCh.ServeSocket(); err != nil {
			Log.Error("Session failure:", err)
		}
	}()

	*socketPath = sshCh.SocketPath()

	<-waitCh
	return nil
}

// WindowChange defines window size change RPC request.
func (rpc *RPCClient) WindowChange(args *WindowChangeArgs, _ *int) error {
	sshCh := rpc.c.chans.getChannel(args.ChannelID)
	if sshCh == nil {
		return errors.New("unknown ssh channel")
	}

	if _, err := sshCh.Request(RequestWindowChange, args.ToBinary()); err != nil {
		return fmt.Errorf("window change request failure: %s", err)
	}

	return nil
}

// Close defines close client RPC request.
func (rpc *RPCClient) Close(channelID *uint32, _ *struct{}) error {
	if rpc == nil {
		return nil
	}

	sshCh := rpc.c.chans.getChannel(*channelID)
	if sshCh == nil {
		return errors.New("unknown ssh channel")
	}

	return sshCh.conn.Close()
}

// RequestPTYArgs defines RequestPTY request parameters.
type RequestPTYArgs struct {
	Username string
	RemotePK cipher.PubKey
	Size     *pty.Winsize
}

// ToBinary returns binary representation of Args.
func (args *RequestPTYArgs) ToBinary() []byte {
	req := appendU32([]byte{}, uint32(args.Size.Cols))
	req = appendU32(req, uint32(args.Size.Rows))
	req = appendU32(req, uint32(args.Size.X))
	req = appendU32(req, uint32(args.Size.Y))
	return append(req, []byte(args.Username)...)
}

// ExecArgs represents Exec response parameters.
type ExecArgs struct {
	ChannelID       uint32
	CommandWithArgs []string
}

// ToBinary returns binary representation of Args.
func (args *ExecArgs) ToBinary() []byte {
	return append([]byte{}, []byte(strings.Join(args.CommandWithArgs, " "))...)
}

// WindowChangeArgs defines WindowChange request parameters.
type WindowChangeArgs struct {
	ChannelID uint32
	Size      *pty.Winsize
}

// ToBinary returns binary representation of Args.
func (args *WindowChangeArgs) ToBinary() []byte {
	req := appendU32([]byte{}, uint32(args.Size.Cols))
	req = appendU32(req, uint32(args.Size.Rows))
	req = appendU32(req, uint32(args.Size.X))
	return appendU32(req, uint32(args.Size.Y))
}
