package apptracker

import (
	"io"
	"log"
	"net"
	"sync"

	"github.com/skycoin/skywire/messages"
)

type AppTracker struct {
	apps           []messages.ServiceInfo
	appsByName     map[string]messages.ServiceInfo
	appsByType     map[string][]messages.ServiceInfo
	address        string
	conn           net.Conn
	opened         bool
	closeChannel   chan bool
	lock           *sync.Mutex
	viscriptServer *ATViscriptServer
}

func NewAppTracker(address string) *AppTracker {
	appTracker := &AppTracker{}
	appTracker.lock = &sync.Mutex{}
	appTracker.address = address
	appTracker.closeChannel = make(chan bool)
	appTracker.apps = []messages.ServiceInfo{}
	appTracker.appsByName = make(map[string]messages.ServiceInfo)
	appTracker.appsByType = make(map[string][]messages.ServiceInfo)
	appTracker.opened = true
	appTracker.serve()
	return appTracker
}

func (self *AppTracker) Shutdown() {
	self.opened = false
}

func (self *AppTracker) serve() {
	address := self.address

	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	log.Println("Apptracker is listening for incoming messages on", self.address)

	go func() {
		conn, err := l.Accept() // accept a connection which is created by orchestration server
		if err != nil {
			log.Println("Cannot accept client's connection:", err)
			return
		}
		defer conn.Close()

		self.conn = conn

		remoteAddr := conn.RemoteAddr().String()

		for {
			buffer := make([]byte, 16384)

			n, err := conn.Read(buffer)
			if err != nil {
				return
				if err == io.EOF {
					continue
				} else {
					log.Printf("error while reading message from %s: %s\n", remoteAddr, err)
					break
				}
			}

			go self.handleIncomingMessage(buffer[:n])
		}
	}()
}

func (self *AppTracker) handleIncomingMessage(msgS []byte) {

	msg := &messages.ServiceRequest{}
	err := messages.Deserialize(msgS, msg)
	if err != nil {
		return
	}

	sequence := msg.Sequence
	payload := msg.Payload

	switch messages.GetMessageType(payload) {

	case messages.MsgAppRegistrationRequest:
		m0 := &messages.AppRegistrationRequest{}
		err := messages.Deserialize(payload, m0)
		if err == nil {
			err = self.registerService(m0)
			self.sendAppRegistrationResponse(err, sequence)
		}

	case messages.MsgAppListRequest:
		m0 := &messages.AppListRequest{}
		err := messages.Deserialize(payload, m0)
		if err == nil {
			apps := self.getServices(m0)
			self.sendAppListResponse(apps, sequence)
		}
	}
}

func (self *AppTracker) sendAppRegistrationResponse(e error, sequence uint32) {
	var eText string

	isError := e != nil
	if isError {
		eText = e.Error()
	}

	resp := messages.AppRegistrationResponse{
		!isError,
		eText,
	}

	respS := messages.Serialize(messages.MsgAppRegistrationResponse, resp)

	self.sendToOrchServer(respS, sequence)
}

func (self *AppTracker) sendAppListResponse(apps []messages.ServiceInfo, sequence uint32) {
	resp := messages.AppListResponse{
		apps,
	}
	respS := messages.Serialize(messages.MsgAppListResponse, resp)
	self.sendToOrchServer(respS, sequence)
}

func (self *AppTracker) sendToOrchServer(payload []byte, sequence uint32) {
	msg := messages.ServiceResponse{
		payload,
		sequence,
	}
	msgS := messages.Serialize(messages.MsgServiceResponse, msg)
	self.conn.Write(msgS)
}

func (self *AppTracker) registerService(msg *messages.AppRegistrationRequest) error {
	serviceInfo := msg.ServiceInfo
	serviceName := serviceInfo.Name
	serviceType := serviceInfo.Type

	self.lock.Lock()
	defer self.lock.Unlock()

	if _, ok := self.appsByName[serviceName]; ok {
		return messages.ERR_SERVICE_EXISTS
	}

	self.apps = append(self.apps, serviceInfo)
	self.appsByName[serviceName] = serviceInfo
	self.appsByType[serviceType] = append(self.appsByType[serviceType], serviceInfo)

	return nil
}

func (self *AppTracker) getServices(msg *messages.AppListRequest) []messages.ServiceInfo {
	t := msg.RequestType
	param := msg.RequestParam

	switch t {
	case "all":
		return self.apps

	case "by_name":
		self.lock.Lock()
		defer self.lock.Unlock()
		result, ok := self.appsByName[param]
		if !ok {
			return []messages.ServiceInfo{}
		}
		return []messages.ServiceInfo{result}

	case "by_type":
		self.lock.Lock()
		defer self.lock.Unlock()
		result := self.appsByType[param]
		return result

	default:
		return []messages.ServiceInfo{}
	}

	return []messages.ServiceInfo{}
}
