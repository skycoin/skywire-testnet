package therealssh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
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
	// RequestWindowChange represents request for PTY size change.
	RequestWindowChange
)

const (
	// ResponseFail represents failed response.
	ResponseFail byte = iota
	// ResponseConfirm represents successful response.
	ResponseConfirm
)

var responseUnauthorised = append([]byte{ResponseFail}, []byte("unauthorised")...)

// Server handles remote PTY data exchange.
type Server struct {
	auth  Authorizer
	chans *chanList
}

// NewServer constructs new Server.
func NewServer(auth Authorizer) *Server {
	return &Server{auth, newChanList()}
}

// OpenChannel opens new client channel.
func (s *Server) OpenChannel(remoteAddr *app.Addr, remoteID uint32, conn net.Conn) error {
	debug("opening new channel")
	channel := OpenChannel(remoteID, remoteAddr, conn)
	var res []byte

	if s.auth.Authorize(remoteAddr.PubKey) != nil {
		res = responseUnauthorised
	} else {
		res = appendU32([]byte{ResponseConfirm}, s.chans.add(channel))
	}

	debug("sending response")
	if err := channel.Send(CmdChannelOpenResponse, res); err != nil {
		channel.Close()
		return fmt.Errorf("channel response failure: %s", err)
	}

	go func() {
		debug("listening for channel requests")
		if err := channel.Serve(); err != nil {
			log.Println("channel failure:", err)
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
		if err := channel.Send(CmdChannelResponse, responseUnauthorised); err != nil {
			log.Println("failed to send response: ", err)
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
		return errors.New("unauthorised")
	}

	if channel.session == nil {
		return errors.New("session is not started")
	}

	channel.dataCh <- data
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

		raddr := conn.RemoteAddr().(*app.Addr)
		payload := buf[:n]

		if len(payload) < 5 {
			return errors.New("corrupted payload")
		}

		payloadID := binary.BigEndian.Uint32(payload[1:])
		data := payload[5:]

		debug("got new command: %x", payload[0])
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
	for _, channel := range s.chans.dropAll() {
		channel.Close()
	}

	return nil
}
