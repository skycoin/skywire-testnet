package factory

import (
	"net"

	"github.com/skycoin/net/client"
	"github.com/skycoin/net/server"
)

type TCPFactory struct {
	listener *net.TCPListener

	FactoryCommonFields
}

func NewTCPFactory() *TCPFactory {
	return &TCPFactory{FactoryCommonFields: NewFactoryCommonFields()}
}

func (factory *TCPFactory) Listen(address string) error {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	factory.fieldsMutex.Lock()
	factory.listener = ln
	factory.fieldsMutex.Unlock()
	go func() {
		for {
			c, err := ln.AcceptTCP()
			if err != nil {
				return
			}
			factory.createConn(c)
		}
	}()
	return nil
}

func (factory *TCPFactory) Close() error {
	factory.FactoryCommonFields.Close()
	factory.fieldsMutex.RLock()
	defer factory.fieldsMutex.RUnlock()
	if factory.listener == nil {
		return nil
	}
	return factory.listener.Close()
}

func (factory *TCPFactory) createConn(c *net.TCPConn) *Connection {
	tcpConn := server.NewServerTCPConn(c)
	tcpConn.SetStatusToConnected()
	conn := newConnection(tcpConn, factory)
	conn.SetContextLogger(conn.GetContextLogger().WithField("type", "tcp"))
	factory.AddAcceptedConn(conn)
	go factory.AcceptedCallback(conn)
	return conn
}

func (factory *TCPFactory) Connect(address string) (conn *Connection, err error) {
	c, err := net.Dial("tcp", address)
	if err != nil {
		return
	}
	cn := client.NewClientTCPConn(c)
	cn.SetStatusToConnected()
	conn = newConnection(cn, factory)
	conn.SetContextLogger(conn.GetContextLogger().WithField("type", "tcp"))
	factory.AddConn(conn)
	return
}
