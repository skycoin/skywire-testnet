package app2

import (
	"errors"
	"net"

	"github.com/skycoin/skywire/pkg/routing"
)

const (
	listenerBufSize = 1000
)

var (
	ErrListenerClosed = errors.New("listener closed")
)

type Listener struct {
	addr  routing.Addr
	conns chan net.Conn
	lm    *listenersManager
}

func NewListener(addr routing.Addr, lm *listenersManager) *Listener {
	return &Listener{
		addr:  addr,
		conns: make(chan net.Conn, listenerBufSize),
		lm:    lm,
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
	if err := l.lm.remove(l.addr.Port); err != nil {
		return err
	}

	// TODO: send ListenEnd frame
	close(l.conns)

	return nil
}

func (l *Listener) Addr() net.Addr {
	return l.addr
}

func (l *Listener) addConn(conn net.Conn) {
	l.conns <- conn
}
