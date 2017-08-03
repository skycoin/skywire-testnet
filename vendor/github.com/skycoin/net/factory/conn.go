package factory

import "github.com/skycoin/net/conn"

type Connection struct {
	conn.Connection
	factory Factory
}
