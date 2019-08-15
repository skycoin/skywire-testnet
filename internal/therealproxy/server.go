package therealproxy

import (
	"fmt"
	"net"

	"github.com/armon/go-socks5"
	"github.com/hashicorp/yamux"
)

// Server implements multiplexing proxy server using yamux.
type Server struct {
	socks    *socks5.Server
	listener net.Listener
}

// NewServer constructs a new Server.
func NewServer(passcode string) (*Server, error) {
	var credentials socks5.CredentialStore
	if passcode != "" {
		credentials = passcodeCredentials(passcode)
	}

	s, err := socks5.New(&socks5.Config{Credentials: credentials})
	if err != nil {
		return nil, fmt.Errorf("socks5: %s", err)
	}

	return &Server{socks: s}, nil
}

// Serve accept connections from listener and serves socks5 proxy for
// the incoming connections.
func (s *Server) Serve(l net.Listener) error {
	s.listener = l
	for {
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("proxyserver Accept: %s", err)
		}

		session, err := yamux.Server(conn, nil)
		if err != nil {
			return fmt.Errorf("yamux: %s", err)
		}

		go func() {
			if err := s.socks.Serve(session); err != nil {
				Logger.Error("Failed to start SOCKS5 server:", err)
			}
		}()
	}
}

// Close implement io.Closer.
func (s *Server) Close() error {
	if s == nil {
		return nil
	}
	return s.listener.Close()
}

type passcodeCredentials string

func (s passcodeCredentials) Valid(user, password string) bool {
	return user == string(s) || password == string(s)
}
