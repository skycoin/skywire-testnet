package app2

import (
	"net"

	"github.com/pkg/errors"

	"github.com/skycoin/skywire/pkg/routing"

	"github.com/skycoin/dmsg/cipher"
)

var (
	ErrWrongHSFrameTypeReceived = errors.New("received wrong HS frame type")
)

type Listener struct {
	conn net.Conn
}

func (l *Listener) Accept() (net.Conn, error) {
	hsFrame, err := readHSFrame(l.conn)
	if err != nil {
		return nil, errors.Wrap(err, "error reading HS frame")
	}

	if hsFrame.FrameType() != HSFrameTypeDMSGAccept {
		return nil, ErrWrongHSFrameTypeReceived
	}

	return l.conn, nil
}

// Client is used by skywire apps.
type Client struct {
	PK       cipher.PubKey
	pid      ProcID
	sockAddr string
	conn     net.Conn
}

// NewClient creates a new Client. The Client needs to be provided with:
// - localPK: The local public key of the parent skywire visor.
// - pid: The procID assigned for the process that Client is being used by.
// - sockAddr: The socket address to connect to Server.
func NewClient(localPK cipher.PubKey, pid ProcID, sockAddr string) (*Client, error) {
	return &Client{
		PK:       localPK,
		pid:      pid,
		sockAddr: sockAddr,
	}, nil
}

func (c *Client) Dial(addr routing.Addr) (net.Conn, error) {
	conn, err := net.Dial("unix", c.sockAddr)
	if err != nil {
		return nil, errors.Wrap(err, "error connecting app server")
	}

	hsFrame := NewHSFrameDSMGDial(c.pid, routing.Loop{
		Local: routing.Addr{
			PubKey: c.PK,
		},
		Remote: addr,
	})
	if _, err := conn.Write(hsFrame); err != nil {
		return nil, errors.Wrap(err, "error writing HS frame")
	}

	hsFrame, err = readHSFrame(conn)
	if err != nil {
		return nil, errors.Wrap(err, "error reading HS frame")
	}

	if hsFrame.FrameType() != HSFrameTypeDMSGAccept {
		return nil, ErrWrongHSFrameTypeReceived
	}

	return conn, nil
}

func (c *Client) Listen(addr routing.Addr) (*Listener, error) {
	conn, err := net.Dial("unix", c.sockAddr)
	if err != nil {
		return nil, errors.Wrap(err, "error connecting app server")
	}

	hsFrame := NewHSFrameDMSGListen(c.pid, addr)
	if _, err := conn.Write(hsFrame); err != nil {
		return nil, errors.Wrap(err, "error writing HS frame")
	}

	hsFrame, err = readHSFrame(conn)
	if err != nil {
		return nil, errors.Wrap(err, "error reading HS frame")
	}

	if hsFrame.FrameType() != HSFrameTypeDMSGListening {
		return nil, ErrWrongHSFrameTypeReceived
	}

	return &Listener{
		conn: conn,
	}, nil
}
