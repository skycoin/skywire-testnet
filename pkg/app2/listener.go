package app2

import (
	"errors"
	"net"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skywire/pkg/routing"
)

const (
	listenerBufSize = 1000
)

var (
	ErrListenerClosed = errors.New("listener closed")
)

type listener struct {
	addr          routing.Addr
	conns         chan *clientConn
	stopListening func(port routing.Port) error
	logger        *logging.Logger
	lm            *listenersManager
	procID        ProcID
}

func newListener(addr routing.Addr, lm *listenersManager, procID ProcID,
	stopListening func(port routing.Port) error, l *logging.Logger) *listener {
	return &listener{
		addr:          addr,
		conns:         make(chan *clientConn, listenerBufSize),
		lm:            lm,
		stopListening: stopListening,
		logger:        l,
		procID:        procID,
	}
}

func (l *listener) Accept() (net.Conn, error) {
	conn, ok := <-l.conns
	if !ok {
		return nil, ErrListenerClosed
	}

	hsFrame := NewHSFrameDMSGAccept(l.procID, routing.Loop{
		Local:  l.addr,
		Remote: conn.remote,
	})

	if _, err := conn.Write(hsFrame); err != nil {
		return nil, err
	}

	return conn, nil
}

func (l *listener) Close() error {
	if err := l.stopListening(l.addr.Port); err != nil {
		l.logger.WithError(err).Error("error sending DmsgStopListening")
	}

	if err := l.lm.remove(l.addr.Port); err != nil {
		return err
	}

	close(l.conns)

	return nil
}

func (l *listener) Addr() net.Addr {
	return l.addr
}

func (l *listener) addConn(conn *clientConn) {
	l.conns <- conn
}
