package app

import (
	"github.com/skycoin/skywire/pkg/routing"
	"io"
	"net"
)

func NewAppMock(conn net.Conn) *App {
	app := &App{proto: NewProtocol(conn), acceptChan: make(chan [2]routing.Addr),
		doneChan: make(chan struct{}),
		conns: make(map[routing.Loop]io.ReadWriteCloser)}
	go app.handleProto()

	return app
}
