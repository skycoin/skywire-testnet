package therealssh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"strings"

	"github.com/kr/pty"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
)

// Client proxies CLI's requests to a remote server. Control messages
// are sent via RPC interface. PTY data is exchanged via unix socket.
type Client struct {
	dial  app.DialFunc
	chans *chanList
}

// NewClient construct new RPC listener and Client from a given RPC address and app dialer.
func NewClient(rpcAddr string, d app.DialFunc) (net.Listener, *Client, error) {
	client := &Client{chans: newChanList(), dial: d}
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
func (c *Client) OpenChannel(remotePK cipher.PubKey) (localID uint32, channel *Channel, cErr error) {
	conn, err := c.dial(app.LoopAddr{PubKey: remotePK, Port: Port})
	if err != nil {
		cErr = fmt.Errorf("dial failed: %s", err)
		return
	}

	channel = OpenClientChannel(0, remotePK, conn)
	debug("sending channel open command")
	localID = c.chans.add(channel)
	req := appendU32([]byte{byte(CmdChannelOpen)}, localID)
	if _, err := conn.Write(req); err != nil {
		cErr = fmt.Errorf("failed to send open channel request: %s", err)
		return
	}

	go func() {
		if err := c.serveConn(conn); err != nil {
			log.Println(err)
		}
	}()

	debug("waiting for channel open response")
	data := <-channel.msgCh
	debug("got channel open response")
	if data[0] == ResponseFail {
		cErr = fmt.Errorf("failed to open channel: %s", string(data[1:]))
		return
	}

	channel.RemoteID = binary.BigEndian.Uint32(data[1:])
	return localID, channel, cErr
}

func (c *Client) resolveChannel(remotePK cipher.PubKey, localID uint32) (*Channel, error) {
	channel := c.chans.getChannel(localID)
	if channel == nil {
		return nil, errors.New("channel is not opened")
	}

	if channel.RemoteAddr.PubKey != remotePK {
		return nil, errors.New("unauthorized")
	}

	return channel, nil
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

		raddr := conn.RemoteAddr().(*app.LoopAddr)
		payload := buf[:n]

		if len(payload) < 5 {
			return errors.New("corrupted payload")
		}

		channel, err := c.resolveChannel(raddr.PubKey, binary.BigEndian.Uint32(payload[1:]))
		if err != nil {
			return err
		}

		data := payload[5:]
		debug("got new command: %x", payload[0])
		switch CommandType(payload[0]) {
		case CmdChannelOpenResponse, CmdChannelResponse:
			channel.msgCh <- data
		case CmdChannelData:
			channel.dataCh <- data
		case CmdChannelServerClose:
			err = channel.Close()
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
	for _, channel := range c.chans.dropAll() {
		channel.Close()
	}

	return nil
}

// RPCClient exposes Client's methods via RPC interface.
type RPCClient struct {
	c *Client
}

// RequestPTY defines RPC request for a new PTY session.
func (rpc *RPCClient) RequestPTY(args *RequestPTYArgs, channelID *uint32) error {
	debug("requesting SSH channel")
	localID, channel, err := rpc.c.OpenChannel(args.RemotePK)
	if err != nil {
		return err
	}

	debug("requesting PTY session")
	if _, err := channel.Request(RequestPTY, args.ToBinary()); err != nil {
		return fmt.Errorf("PTY request failure: %s", err)
	}

	*channelID = localID
	return nil
}

// Exec defines new remote execution RPC request.
func (rpc *RPCClient) Exec(args *ExecArgs, socketPath *string) error {
	channel := rpc.c.chans.getChannel(args.ChannelID)
	if channel == nil {
		return errors.New("unknown channel")
	}

	debug("requesting shell process")
	if args.CommandWithArgs == nil {
		if _, err := channel.Request(RequestShell, nil); err != nil {
			return fmt.Errorf("Shell request failure: %s", err)
		}
	} else {
		if _, err := channel.Request(RequestExec, args.ToBinary()); err != nil {
			return fmt.Errorf("Shell request failure: %s", err)
		}
	}

	waitCh := make(chan bool)
	go func() {
		debug("starting socket listener")
		waitCh <- true
		if err := channel.ServeSocket(); err != nil {
			log.Println("Session failure:", err)
		}
	}()

	*socketPath = channel.SocketPath()
	<-waitCh
	return nil
}

// WindowChange defines window size change RPC request.
func (rpc *RPCClient) WindowChange(args *WindowChangeArgs, _ *int) error {
	channel := rpc.c.chans.getChannel(args.ChannelID)
	if channel == nil {
		return errors.New("unknown channel")
	}

	if _, err := channel.Request(RequestWindowChange, args.ToBinary()); err != nil {
		return fmt.Errorf("window change request failure: %s", err)
	}

	return nil
}

// Close defines close client RPC request.
func (rpc *RPCClient) Close(channelID *uint32, _ *struct{}) error {
	channel := rpc.c.chans.getChannel(*channelID)
	if channel == nil {
		return errors.New("unknown channel")
	}

	return channel.conn.Close()
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
