package therealssh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
)

// CommandType represents global protocol messages.
type CommandType byte

const (
	// CmdChannelOpen represents channel open message.
	CmdChannelOpen CommandType = iota
	// CmdChannelOpenResponse represents channel open response message.
	CmdChannelOpenResponse
	// CmdChannelRequest represents request message.
	CmdChannelRequest
	// CmdChannelResponse represents response message.
	CmdChannelResponse
	// CmdChannelData represents data message.
	CmdChannelData
	// CmdChannelServerClose represents server message about closing channel.
	CmdChannelServerClose
)

// RequestType represents channel requests types.
type RequestType byte

const (
	// RequestPTY represents request for new PTY session.
	RequestPTY RequestType = iota
	// RequestShell represents request for new shell.
	RequestShell
	// RequestExec represents request for new process.
	RequestExec
	// RequestExecWithoutShell for use in integration testing.
	RequestExecWithoutShell
	// RequestWindowChange represents request for PTY size change.
	RequestWindowChange
)

const (
	// ResponseFail represents failed response.
	ResponseFail byte = iota
	// ResponseConfirm represents successful response.
	ResponseConfirm
)

var responseUnauthorized = append([]byte{ResponseFail}, []byte("unauthorized")...)

// Server handles remote PTY data exchange.
type Server struct {
	log   *logging.Logger
	auth  Authorizer
	chans *chanList
}

// NewServer constructs new Server.
func NewServer(auth Authorizer, log *logging.MasterLogger) *Server {
	return &Server{log.PackageLogger("therealssh_server"), auth, newChanList()}
}

// OpenChannel opens new client channel.
func (s *Server) OpenChannel(remoteAddr routing.Addr, remoteID uint32, conn net.Conn) error {
	Log.Debugln("opening new channel")
	channel := OpenChannel(remoteID, remoteAddr, conn)
	var res []byte

	if s.auth.Authorize(remoteAddr.PubKey) != nil {
		res = responseUnauthorized
	} else {
		res = appendU32([]byte{ResponseConfirm}, s.chans.add(channel))
	}

	s.log.Debugln("sending response")
	if err := channel.Send(CmdChannelOpenResponse, res); err != nil {
		if err := channel.Close(); err != nil {
			Log.WithError(err).Warn("Failed to close channel")
		}
		return fmt.Errorf("channel response failure: %s", err)
	}

	go func() {
		s.log.Debugln("listening for channel requests")
		if err := channel.Serve(); err != nil {
			Log.Error("channel failure:", err)
		}
	}()

	return nil
}

// HandleRequest implements multiplexing logic for request messages.
func (s *Server) HandleRequest(remotePK cipher.PubKey, localID uint32, data []byte) error {
	channel := s.chans.getChannel(localID)
	if channel == nil {
		return errors.New("channel is not opened")
	}

	if s.auth.Authorize(remotePK) != nil || channel.RemoteAddr.PubKey != remotePK {
		if err := channel.Send(CmdChannelResponse, responseUnauthorized); err != nil {
			Log.Error("failed to send response: ", err)
		}
		return nil
	}

	channel.msgCh <- data
	return nil
}

// HandleData implements multiplexing logic for data messages.
func (s *Server) HandleData(remotePK cipher.PubKey, localID uint32, data []byte) error {
	channel := s.chans.getChannel(localID)
	if channel == nil {
		return errors.New("channel is not opened")
	}

	if s.auth.Authorize(remotePK) != nil || channel.RemoteAddr.PubKey != remotePK {
		return errors.New("unauthorized")
	}

	if channel.session == nil {
		return errors.New("session is not started")
	}

	channel.dataChMx.Lock()
	if !channel.IsClosed() {
		channel.dataCh <- data
	}
	channel.dataChMx.Unlock()
	return nil
}

// Serve defines routing rules for received App messages.
func (s *Server) Serve(conn net.Conn) error {
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

		payloadID := binary.BigEndian.Uint32(payload[1:])
		data := payload[5:]

		s.log.Debugf("got new command: %x", payload[0])
		switch CommandType(payload[0]) {
		case CmdChannelOpen:
			err = s.OpenChannel(raddr, payloadID, conn)
		case CmdChannelRequest:
			err = s.HandleRequest(raddr.PubKey, payloadID, data)
		case CmdChannelData:
			err = s.HandleData(raddr.PubKey, payloadID, data)
		default:
			err = fmt.Errorf("unknown command: %x", payload[0])
		}

		if err != nil {
			return err
		}
	}
}

// Close closes all opened channels.
func (s *Server) Close() error {
	if s == nil {
		return nil
	}

	for _, channel := range s.chans.dropAll() {
		if err := channel.Close(); err != nil {
			Log.WithError(err).Warn("Failed to close channel")
		}
	}

	return nil
}
