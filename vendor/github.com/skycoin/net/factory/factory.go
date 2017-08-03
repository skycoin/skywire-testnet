package factory

import "sync"

type Factory interface {
	Close() error
}

type FactoryCommonFields struct {
	AcceptedCallback func(connection *Connection)

	connections      map[*Connection]struct{}
	connectionsMutex sync.RWMutex

	fieldsMutex sync.RWMutex
}

func NewFactoryCommonFields() FactoryCommonFields {
	return FactoryCommonFields{connections: make(map[*Connection]struct{})}
}

func (f *FactoryCommonFields) AddConn(conn *Connection) {
	f.connectionsMutex.Lock()
	f.connections[conn] = struct{}{}
	f.connectionsMutex.Unlock()
	go func() {
		conn.WriteLoop()
		f.RemoveConn(conn)
	}()
	go conn.ReadLoop()
}

func (f *FactoryCommonFields) RemoveConn(conn *Connection) {
	f.connectionsMutex.Lock()
	delete(f.connections, conn)
	f.connectionsMutex.Unlock()
}
