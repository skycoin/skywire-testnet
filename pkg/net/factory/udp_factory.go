package factory

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/SkycoinProject/skywire/pkg/net/msg"

	"github.com/SkycoinProject/skywire/pkg/net/client"
	"github.com/SkycoinProject/skywire/pkg/net/conn"
	"github.com/SkycoinProject/skywire/pkg/net/server"
)

type UDPFactory struct {
	listener *net.UDPConn
	server   *server.ServerUDPConn

	FactoryCommonFields

	udpConnMapMutex sync.RWMutex
	udpConnMap      map[string]*Connection

	stopGC chan struct{}

	BeforeReadOnConn func(m *msg.UDPMessage)
	BeforeSendOnConn func(m *msg.UDPMessage)
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
	udpConn.BeforeRead = factory.BeforeReadOnConn
	udpConn.BeforeSend = factory.BeforeSendOnConn
	udpConn.SetStatusToConnected()
	connection := newConnection(udpConn, factory)
	factory.udpConnMap[addr.String()] = connection
	factory.udpConnMapMutex.Unlock()

	connection.SetContextLogger(connection.GetContextLogger().WithField("type", "udp").
		WithField("addr", addr.String()).
		WithField("udp_factory", fmt.Sprintf("%p", factory)).
		WithField("dir", "in"))
	factory.AddAcceptedConn(connection)
	go factory.AcceptedCallback(connection)
	return udpConn
}

func (factory *UDPFactory) createConnAfterListen(addr *net.UDPAddr, skipBeforeCallbacks bool) (*Connection, bool) {
	factory.udpConnMapMutex.Lock()
	if cc, ok := factory.udpConnMap[addr.String()]; ok {
		factory.udpConnMapMutex.Unlock()
		return cc, false
	}

	factory.fieldsMutex.Lock()
	ln := factory.listener
	factory.fieldsMutex.Unlock()

	udpConn := conn.NewUDPConn(ln, addr)
	if !skipBeforeCallbacks {
		udpConn.BeforeRead = factory.BeforeReadOnConn
		udpConn.BeforeSend = factory.BeforeSendOnConn
	}
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
					udp.SetStatusToError(errors.New("udp gc timeout"))
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
	conn.SetContextLogger(conn.GetContextLogger().
		WithField("type", "udp").
		WithField("udp_factory", fmt.Sprintf("%p", factory)).
		WithField("dir", "out"))
	factory.AddConn(conn)
	return
}

func (factory *UDPFactory) ConnectAfterListen(address string, skipBeforeCallbacks bool) (conn *Connection, err error) {
	ra, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}
	conn, create := factory.createConnAfterListen(ra, skipBeforeCallbacks)
	if !create {
		return nil, nil
	}
	conn.SetContextLogger(conn.GetContextLogger().
		WithField("type", "udpe").
		WithField("udp_factory", fmt.Sprintf("%p", factory)).
		WithField("addr", address))
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
