package servicemanager

import (
	"log"
	"net"
	"sync"

	"github.com/skycoin/skywire/src/messages"
)

type ServiceManager struct {
	servicesByName map[string]string
	servicesByType map[string][]string
	conn           net.Conn
	closeChannel   chan bool
	lock           *sync.Mutex
}

func NewServiceManager(orchAddr string) (*ServiceManager, error) {
	serviceManager := &ServiceManager{}
	serviceManager.lock = &sync.Mutex{}
	conn, err := net.Dial("tcp", orchAddr)
	if err != nil {
		return nil, err
	}
	serviceManager.conn = conn
	serviceManager.closeChannel = make(chan bool)
	return serviceManager, nil
}

func (self *ServiceManager) Shutdown() {
	self.closeChannel <- true
}

func (self *ServiceManager) Serve() {
	go_on := true
	go func() {
		for go_on {

			buffer := make([]byte, 512)

			n, err := self.conn.Read(buffer)

			if err != nil {
				if !go_on && n == 0 {
					break
				} else {
					panic(err)
				}
			} else {
				if n > 0 {
					log.Printf("connection at %s received %d bytes\n", self.conn.LocalAddr().String(), n)
					go self.handleIncomingMessage(buffer[:n])
				}
			}
		}
	}()
	<-self.closeChannel
	go_on = false
	self.conn.Close()
}

func (self *ServiceManager) handleIncomingMessage(msgS []byte) {
	switch messages.GetMessageType(msgS) {

	case messages.MsgServiceRegistrationRequest:
		msg := &messages.ServiceRegistrationRequest{}
		err := messages.Deserialize(msgS, msg)
		if err != nil {
			self.sendServiceRegistrationResponse(err)
		}

		err = self.registerService(msg)
		self.sendServiceRegistrationResponse(err)

	case messages.MsgServiceRequest:
		msg := &messages.ServiceRequest{}
		err := messages.Deserialize(msgS, msg)
		if err != nil {
			self.sendServiceResponse([]messages.ServiceInfo{}, err)
		}

		services, err := self.getServices(msg)
		self.sendServiceResponse(services, err)
	}
}

func (self *ServiceManager) sendServiceRegistrationResponse(err error) {
	// do something
}

func (self *ServiceManager) sendServiceResponse(services []messages.ServiceInfo, err error) {
	// do something
}

func (self *ServiceManager) registerService(msg *messages.ServiceRegistrationRequest) error {
	//do something
	return nil
}

func (self *ServiceManager) getServices(msg *messages.ServiceRequest) ([]messages.ServiceInfo, error) {
	// do something
	return []messages.ServiceInfo{}, nil
}
