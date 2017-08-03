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

	FactoryCommonFields

	udpConnMapMutex sync.RWMutex
	udpConnMap      map[string]*conn.UDPConn

	stopGC chan bool
}

func NewUDPFactory() *UDPFactory {
	udpFactory := &UDPFactory{stopGC: make(chan bool), FactoryCommonFields: NewFactoryCommonFields(), udpConnMap: make(map[string]*conn.UDPConn)}
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
	factory.fieldsMutex.Unlock()
	go func() {
		udpc := server.NewServerUDPConn(udp)
		udpc.ReadLoop(factory.createConn)
	}()
	return nil
}

func (factory *UDPFactory) Close() error {
	factory.stopGC <- true
	factory.fieldsMutex.RLock()
	defer factory.fieldsMutex.RUnlock()
	if factory.listener == nil {
		return nil
	}
	return factory.listener.Close()
}

func (factory *UDPFactory) createConn(c *net.UDPConn, addr *net.UDPAddr) *conn.UDPConn {
	factory.udpConnMapMutex.RLock()
	if cc, ok := factory.udpConnMap[addr.String()]; ok {
		factory.udpConnMapMutex.RUnlock()
		return cc
	}
	factory.udpConnMapMutex.RUnlock()

	udpConn := conn.NewUDPConn(c, addr)
	factory.udpConnMapMutex.Lock()
	factory.udpConnMap[addr.String()] = udpConn
	factory.udpConnMapMutex.Unlock()

	udpConn.SetStatusToConnected()
	connection := &Connection{Connection: udpConn, factory: factory}
	connection.SetContextLogger(connection.GetContextLogger().WithField("type", "udp"))
	factory.AddConn(connection)
	go factory.AcceptedCallback(connection)
	return udpConn
}

const UDP_GC_PERIOD = 90

func (factory *UDPFactory) GC() {
	ticker := time.NewTicker(time.Second * UDP_GC_PERIOD)
	for {
		select {
		case <-factory.stopGC:
			return
		case <-ticker.C:
			nowUnix := time.Now().Unix()
			closed := []string{}
			factory.udpConnMapMutex.RLock()
			for k, udp := range factory.udpConnMap {
				if nowUnix-udp.GetLastTime() >= UDP_GC_PERIOD {
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
	c, err := net.Dial("udp", address)
	if err != nil {
		return
	}
	udp := c.(*net.UDPConn)
	cn := client.NewClientUDPConn(udp)
	cn.SetStatusToConnected()
	conn = &Connection{Connection: cn, factory: factory}
	conn.SetContextLogger(conn.GetContextLogger().WithField("type", "udp"))
	factory.AddConn(conn)
	return
}
