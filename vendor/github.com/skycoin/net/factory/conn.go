package factory

import "github.com/skycoin/net/conn"

type Connection struct {
	conn.Connection
	factory    Factory
	RealObject interface{}
}

func newConnection(connection conn.Connection, factory Factory) (c *Connection) {
	c = &Connection{Connection: connection, factory: factory}
	return
}
