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

type Listener struct {
	addr          routing.Addr
	conns         chan net.Conn
	stopListening func(port routing.Port) error
	logger        *logging.Logger
	lm            *listenersManager
}

func NewListener(addr routing.Addr, lm *listenersManager, stopListening func(port routing.Port) error, l *logging.Logger) *Listener {
	return &Listener{
		addr:          addr,
		conns:         make(chan net.Conn, listenerBufSize),
		lm:            lm,
		stopListening: stopListening,
		logger:        l,
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.conns
	if !ok {
		return nil, ErrListenerClosed
	}

	return conn, nil
}

func (l *Listener) Close() error {
	if err := l.stopListening(l.addr.Port); err != nil {
		l.logger.WithError(err).Error("error sending DmsgStopListening")
	}

	if err := l.lm.remove(l.addr.Port); err != nil {
		return err
	}

	close(l.conns)

	return nil
}

func (l *Listener) Addr() net.Addr {
	return l.addr
}

func (l *Listener) addConn(conn net.Conn) {
	l.conns <- conn
}
