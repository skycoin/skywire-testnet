package app2

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/hashicorp/yamux"

	"github.com/skycoin/dmsg"

	"github.com/skycoin/skywire/pkg/routing"

	"github.com/pkg/errors"

	"github.com/skycoin/dmsg/cipher"
)

type clientConn struct {
	conn    net.Conn
	session *yamux.Session
	lm      *listenersManager
	dmsgL   *dmsg.Listener
}

// Server is used by skywire visor.
type Server struct {
	PK     cipher.PubKey
	dmsgC  *dmsg.Client
	apps   map[string]*clientConn
	appsMx sync.Mutex
	logger *logging.Logger
}

func NewServer(localPK cipher.PubKey, dmsgC *dmsg.Client, l *logging.Logger) *Server {
	return &Server{
		PK:     localPK,
		dmsgC:  dmsgC,
		apps:   make(map[string]*clientConn),
		logger: l,
	}
}

func (s *Server) Serve(sockAddr string) error {
	l, err := net.Listen("unix", sockAddr)
	if err != nil {
		return errors.Wrap(err, "error listening unix socket")
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return errors.Wrap(err, "error accepting client connection")
		}

		s.appsMx.Lock()
		if _, ok := s.apps[conn.RemoteAddr().String()]; ok {
			s.logger.WithError(ErrPortAlreadyBound).Error("error storing session")
		}

		session, err := yamux.Server(conn, nil)
		if err != nil {
			return errors.Wrap(err, "error creating yamux session")
		}

		s.apps[conn.RemoteAddr().String()] = &clientConn{
			session: session,
			conn:    conn,
			lm:      newListenersManager(),
		}
		s.appsMx.Unlock()

		// TODO: handle error
		go s.serveClient(session)
	}
}

func (s *Server) serveClient(session *yamux.Session) error {
	for {
		stream, err := session.Accept()
		if err != nil {
			return errors.Wrap(err, "error opening stream")
		}

		go s.serveStream(stream)

		hsFrame, err := readHSFrame(conn)
		if err != nil {
			return errors.Wrap(err, "error reading HS frame")
		}

		switch hsFrame.FrameType() {
		case HSFrameTypeDMSGListen:
			pk := make(cipher.PubKey, 33)
			copy(pk, hsFrame[HSFrameHeaderLen:HSFrameHeaderLen+HSFramePKLen])
			port := binary.BigEndian.Uint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:])
			dmsgL, err := s.dmsgC.Listen(port)
			if err != nil {
				return fmt.Errorf("error listening on port %d: %v", port, err)
			}

			respHSFrame := NewHSFrameDMSGListening(hsFrame.ProcID(), routing.Addr{
				PubKey: cipher.PubKey(pk),
				Port:   0,
			})
		}
	}
}

func (s *Server) serveStream(stream net.Conn) error {
	for {
		hsFrame, err := readHSFrame(stream)
		if err != nil {
			return errors.Wrap(err, "error reading HS frame")
		}

		switch hsFrame.FrameType() {
		case HSFrameTypeDMSGListen:
			var pk cipher.PubKey
			copy(pk[:], hsFrame[HSFrameHeaderLen:HSFrameHeaderLen+HSFramePKLen])
			port := binary.BigEndian.Uint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:])
			dmsgL, err := s.dmsgC.Listen(port)
			if err != nil {
				return fmt.Errorf("error listening on port %d: %v", port, err)
			}

			respHSFrame := NewHSFrameDMSGListening(hsFrame.ProcID(), routing.Addr{
				PubKey: pk,
				Port:   routing.Port(port),
			})

			if _, err := stream.Write(respHSFrame); err != nil {
				return errors.Wrap(err, "error writing response")
			}

		}
	}
}

func (s *Server) handleDMSGListen(frame HSFrame) error {
	var local routing.Addr
	if err := frame.UnmarshalBody(&local); err != nil {
		return errors.Wrap(err, "invalid JSON body")
	}

	// TODO: check `local` for validity

	dmsgL, err := s.dmsgC.Listen(uint16(local.Port))
	if err != nil {
		return fmt.Errorf("error listening on port %d: %v", local.Port, err)
	}

}

func (s *Server) handleDMSGListening(frame HSFrame) error {
	var local routing.Addr
	if err := frame.UnmarshalBody(&local); err != nil {
		return errors.Wrap(err, "invalid JSON body")
	}
}

func (s *Server) handleDMSGDial(frame HSFrame) error {
	var loop routing.Loop
	if err := frame.UnmarshalBody(&loop); err != nil {
		return errors.Wrap(err, "invalid JSON body")
	}
}

func (s *Server) handleDMSGAccept(frame HSFrame) error {
	var loop routing.Loop
	if err := frame.UnmarshalBody(&loop); err != nil {
		return errors.Wrap(err, "invalid JSON body")
	}
}
