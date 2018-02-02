package factory

import (
	"net"
	"sync"
	"time"

	"github.com/skycoin/net/client"
	"github.com/skycoin/net/conn"
	"github.com/skycoin/net/server"
)

type UDPFactory struct {
	listener *net.UDPConn
	server   *server.ServerUDPConn

	FactoryCommonFields

	udpConnMapMutex sync.RWMutex
	udpConnMap      map[string]*Connection

	stopGC chan struct{}
}

func NewUDPFactory() *UDPFactory {
	udpFactory := &UDPFactory{
		stopGC:              make(chan struct{}),
		FactoryCommonFields: NewFactoryCommonFields(),
		udpConnMap:          make(map[string]*Connection),
	}
	go udpFactory.GC()
	return udpFactory
}

func (factory *UDPFactory) Listen(address string) error {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}
	udp, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	factory.fieldsMutex.Lock()
	factory.listener = udp
	factory.server = server.NewServerUDPConn(udp)
	factory.fieldsMutex.Unlock()
	go func() {
		factory.server.ReadLoop(factory.createConn)
	}()
	return nil
}

func (factory *UDPFactory) Close() error {
	factory.fieldsMutex.RLock()
	defer factory.fieldsMutex.RUnlock()
	if factory.server == nil {
		return nil
	}
	close(factory.stopGC)
	factory.acceptedConnectionsMutex.RLock()
	for k := range factory.acceptedConnections {
		k.Close()
	}
	factory.acceptedConnectionsMutex.RUnlock()
	factory.FactoryCommonFields.Close()
	factory.server.Close()
	factory.server = nil
	return nil
}

func (factory *UDPFactory) createConn(c *net.UDPConn, addr *net.UDPAddr) *conn.UDPConn {
	factory.udpConnMapMutex.Lock()
	if cc, ok := factory.udpConnMap[addr.String()]; ok {
		factory.udpConnMapMutex.Unlock()
		return cc.Connection.(*conn.UDPConn)
	}

	udpConn := conn.NewUDPConn(c, addr)
	udpConn.SetStatusToConnected()
	connection := newConnection(udpConn, factory)
	factory.udpConnMap[addr.String()] = connection
	factory.udpConnMapMutex.Unlock()

	connection.SetContextLogger(connection.GetContextLogger().WithField("type", "udp").WithField("addr", addr.String()))
	factory.AddAcceptedConn(connection)
	go factory.AcceptedCallback(connection)
	return udpConn
}

func (factory *UDPFactory) createConnAfterListen(addr *net.UDPAddr) (*Connection, bool) {
	factory.udpConnMapMutex.Lock()
	if cc, ok := factory.udpConnMap[addr.String()]; ok {
		factory.udpConnMapMutex.Unlock()
		return cc, false
	}

	factory.fieldsMutex.Lock()
	ln := factory.listener
	factory.fieldsMutex.Unlock()

	udpConn := conn.NewUDPConn(ln, addr)
	udpConn.SendPing = true
	udpConn.SetStatusToConnected()
	connection := newConnection(udpConn, factory)
	factory.udpConnMap[addr.String()] = connection
	factory.udpConnMapMutex.Unlock()
	factory.AddAcceptedConn(connection)
	return connection, true
}

func (factory *UDPFactory) GC() {
	ticker := time.NewTicker(time.Second * conn.UDP_GC_PERIOD)
	for {
		select {
		case <-factory.stopGC:
			return
		case <-ticker.C:
			nowUnix := time.Now().Unix()
			var closed []string
			factory.udpConnMapMutex.RLock()
			for k, udp := range factory.udpConnMap {
				if nowUnix-udp.GetLastTime() >= conn.UDP_GC_PERIOD {
					udp.Close()
					closed = append(closed, k)
				}
			}
			factory.udpConnMapMutex.RUnlock()
			if len(closed) < 1 {
				continue
			}
			factory.udpConnMapMutex.Lock()
			for _, u := range closed {
				delete(factory.udpConnMap, u)
			}
			factory.udpConnMapMutex.Unlock()
		}
	}
}

func (factory *UDPFactory) Connect(address string) (conn *Connection, err error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}
	udp, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return
	}
	cn := client.NewClientUDPConn(udp, addr)
	cn.SetStatusToConnected()
	conn = newConnection(cn, factory)
	conn.SetContextLogger(conn.GetContextLogger().WithField("type", "udp"))
	factory.AddConn(conn)
	return
}

func (factory *UDPFactory) ConnectAfterListen(address string) (conn *Connection, err error) {
	ra, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}
	conn, create := factory.createConnAfterListen(ra)
	if !create {
		return nil, nil
	}
	conn.SetContextLogger(conn.GetContextLogger().WithField("type", "udpe").WithField("addr", address))
	return
}

func (factory *UDPFactory) AddAcceptedConn(conn *Connection) {
	factory.acceptedConnectionsMutex.Lock()
	factory.acceptedConnections[conn] = struct{}{}
	factory.acceptedConnectionsMutex.Unlock()
	go func() {
		conn.WriteLoop()
		factory.RemoveAcceptedConn(conn)
	}()
}

func (factory *UDPFactory) RemoveAcceptedConn(conn *Connection) {
	factory.udpConnMapMutex.Lock()
	delete(factory.udpConnMap, conn.GetRemoteAddr().String())
	factory.udpConnMapMutex.Unlock()
	factory.FactoryCommonFields.RemoveAcceptedConn(conn)
}
