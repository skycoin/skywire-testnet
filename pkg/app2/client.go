package app2

import (
	"encoding/binary"
	"net"

	"github.com/hashicorp/yamux"

	"github.com/pkg/errors"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skywire/pkg/routing"

	"github.com/skycoin/dmsg/cipher"
)

var (
	ErrWrongHSFrameTypeReceived = errors.New("received wrong HS frame type")
)

// Client is used by skywire apps.
type Client struct {
	PK          cipher.PubKey
	pid         ProcID
	sockAddr    string
	conn        net.Conn
	session     *yamux.Session
	logger      *logging.Logger
	lm          *listenersManager
	isListening int32
}

// NewClient creates a new Client. The Client needs to be provided with:
// - localPK: The local public key of the parent skywire visor.
// - pid: The procID assigned for the process that Client is being used by.
// - sockAddr: The socket address to connect to Server.
func NewClient(localPK cipher.PubKey, pid ProcID, sockAddr string, l *logging.Logger) (*Client, error) {
	conn, err := net.Dial("unix", sockAddr)
	if err != nil {
		return nil, errors.Wrap(err, "error connecting app server")
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error opening yamux session")
	}

	lm := newListenersManager(l, pid, localPK)

	return &Client{
		PK:       localPK,
		pid:      pid,
		sockAddr: sockAddr,
		conn:     conn,
		session:  session,
		lm:       lm,
	}, nil
}

func (c *Client) Dial(addr routing.Addr) (net.Conn, error) {
	stream, err := c.session.Open()
	if err != nil {
		return nil, errors.Wrap(err, "error opening stream")
	}

	err = dialHS(stream, c.pid, routing.Loop{
		Local: routing.Addr{
			PubKey: c.PK,
		},
		Remote: addr,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error performing Dial HS")
	}

	return stream, nil
}

func (c *Client) Listen(port routing.Port) (net.Listener, error) {
	if err := c.lm.reserveListener(port); err != nil {
		return nil, errors.Wrap(err, "error reserving listener")
	}

	stream, err := c.session.Open()
	if err != nil {
		return nil, errors.Wrap(err, "error opening stream")
	}

	local := routing.Addr{
		PubKey: c.PK,
		Port:   port,
	}

	err = listenHS(stream, c.pid, local)
	if err != nil {
		return nil, errors.Wrap(err, "error performing Listen HS")
	}

	c.lm.listen(c.session)

	l := newListener(local, c.lm, c.pid, c.stopListening, c.logger)
	if err := c.lm.set(port, l); err != nil {
		return nil, errors.Wrap(err, "error setting listener")
	}

	return l, nil
}

func (c *Client) listen() error {
	for {
		stream, err := c.session.Accept()
		if err != nil {
			return errors.Wrap(err, "error accepting stream")
		}

		hsFrame, err := readHSFrame(stream)
		if err != nil {
			c.logger.WithError(err).Error("error reading HS frame")
			continue
		}

		if hsFrame.FrameType() != HSFrameTypeDMSGDial {
			c.logger.WithError(ErrWrongHSFrameTypeReceived).Error("on listening for Dial")
			continue
		}

		// TODO: handle field get gracefully
		remotePort := routing.Port(binary.BigEndian.Uint16(hsFrame[HSFrameHeaderLen+HSFramePKLen*2+HSFramePortLen:]))
		if err := c.lm.addConn(remotePort, stream); err != nil {
			c.logger.WithError(err).Error("failed to accept")
			continue
		}

		localPort := routing.Port(binary.BigEndian.Uint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:]))

		var localPK cipher.PubKey
		copy(localPK[:], hsFrame[HSFrameHeaderLen:HSFrameHeaderLen+HSFramePKLen])

		respHSFrame := NewHSFrameDMSGAccept(c.pid, routing.Loop{
			Local: routing.Addr{
				PubKey: c.PK,
				Port:   remotePort,
			},
			Remote: routing.Addr{
				PubKey: localPK,
				Port:   localPort,
			},
		})

		if _, err := stream.Write(respHSFrame); err != nil {
			c.logger.WithError(err).Error("error responding with DmsgAccept")
			continue
		}
	}
}

func (c *Client) stopListening(port routing.Port) error {
	stream, err := c.session.Open()
	if err != nil {
		return errors.Wrap(err, "error opening stream")
	}

	addr := routing.Addr{
		PubKey: c.PK,
		Port:   port,
	}

	hsFrame := NewHSFrameDMSGStopListening(c.pid, addr)
	if _, err := stream.Write(hsFrame); err != nil {
		return errors.Wrap(err, "error writing HS frame")
	}

	if err := stream.Close(); err != nil {
		return errors.Wrap(err, "error closing stream")
	}

	return nil
}
