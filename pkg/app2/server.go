package app2

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
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
	conn            net.Conn
	session         *yamux.Session
	lm              *listenersManager
	dmsgListeners   map[routing.Port]*dmsg.Listener
	dmsgListenersMx sync.RWMutex
}

// Server is used by skywire visor.
type Server struct {
	PK     cipher.PubKey
	dmsgC  *dmsg.Client
	apps   map[string]*clientConn
	appsMx sync.RWMutex
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
			session:       session,
			conn:          conn,
			lm:            newListenersManager(),
			dmsgListeners: make(map[routing.Port]*dmsg.Listener),
		}
		s.appsMx.Unlock()

		// TODO: handle error
		go s.serveClient(session)
	}
}

func (s *Server) serveClient(conn *clientConn) error {
	for {
		stream, err := conn.session.Accept()
		if err != nil {
			return errors.Wrap(err, "error opening stream")
		}

		go s.serveStream(stream, conn)
///////////////////////////////
		hsFrame, err := readHSFrame(conn)
		if err != nil {
			return errors.Wrap(err, "error reading HS frame")
		}

		switch hsFrame.FrameType() {
		case HSFrameTypeDMSGListen:
			if s.
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

func (s *Server) serveStream(stream net.Conn, conn *clientConn) error {
	for {
		hsFrame, err := readHSFrame(stream)
		if err != nil {
			return errors.Wrap(err, "error reading HS frame")
		}

		var respHSFrame HSFrame
		switch hsFrame.FrameType() {
		case HSFrameTypeDMSGListen:
			port := binary.BigEndian.Uint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:])
			if err := conn.reserveListener(routing.Port(port)); err != nil {
				respHSFrame = NewHSFrameError(hsFrame.ProcID())
			} else {
				dmsgL, err := s.dmsgC.Listen(port)
				if err != nil {
					respHSFrame = NewHSFrameError(hsFrame.ProcID())
				} else {
					if err := conn.addListener(routing.Port(port), dmsgL); err != nil {
						respHSFrame = NewHSFrameError(hsFrame.ProcID())
					} else {
						var pk cipher.PubKey
						copy(pk[:], hsFrame[HSFrameHeaderLen:HSFrameHeaderLen+HSFramePKLen])

						respHSFrame = NewHSFrameDMSGListening(hsFrame.ProcID(), routing.Addr{
							PubKey: pk,
							Port:   routing.Port(port),
						})
					}
				}
			}
		case HSFrameTypeDMSGDial:
			localPort := binary.BigEndian.Uint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:])
			var localPK cipher.PubKey
			copy(localPK[:], hsFrame[HSFrameHeaderLen:HSFrameHeaderLen+HSFramePKLen])

			var remotePK cipher.PubKey
			copy(remotePK[:], hsFrame[HSFrameHeaderLen+HSFramePKLen+HSFramePortLen:HSFrameHeaderLen+HSFramePKLen+HSFramePortLen+HSFramePKLen])
			remotePort := binary.BigEndian.Uint16(hsFrame[HSFrameHeaderLen+HSFramePKLen+HSFramePortLen+HSFramePKLen:])

			// TODO: context
			tp, err := s.dmsgC.Dial(context.Background(), localPK, localPort)
			if err != nil {
				respHSFrame = NewHSFrameError(hsFrame.ProcID())
			} else {
				respHSFrame = NewHSFrameDMSGAccept(hsFrame.ProcID(), routing.Loop{
					Local: routing.Addr{
						PubKey: localPK,
						Port: routing.Port(localPort),
					},
					Remote: routing.Addr{
						PubKey: remotePK,
						Port: routing.Port(remotePort),
					},
				})

				go func() {
					if err := s.forwardOverDMSG(stream, tp); err != nil {
						s.logger.WithError(err).Error("error forwarding over DMSG")
					}
				}()
			}
		}

		if _, err := stream.Write(respHSFrame); err != nil {
			return errors.Wrap(err, "error writing response")
		}
	}
}

func (s *Server) forwardOverDMSG(stream net.Conn, tp *dmsg.Transport) error {
	toStreamErrCh := make(chan error)
	defer close(toStreamErrCh)
	go func() {
		_, err := io.Copy(stream, tp)
		toStreamErrCh <- err
	}()

	_, err := io.Copy(stream, tp)
	if err != nil {
		return err
	}

	if err := <-toStreamErrCh; err != nil {
		return err
	}

	return nil
}

func (c *clientConn) reserveListener(port routing.Port) error {
	c.dmsgListenersMx.Lock()
	if _, ok := c.dmsgListeners[port]; ok {
		c.dmsgListenersMx.Unlock()
		return ErrPortAlreadyBound
	}
	c.dmsgListeners[port] = nil
	c.dmsgListenersMx.Unlock()
	return nil
}

func (c *clientConn) addListener(port routing.Port, l *dmsg.Listener) error {
	c.dmsgListenersMx.Lock()
	if lis, ok := c.dmsgListeners[port]; ok && lis != nil {
		c.dmsgListenersMx.Unlock()
		return ErrPortAlreadyBound
	}
	c.dmsgListeners[port] = l
	c.dmsgListenersMx.Unlock()
	return nil
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
