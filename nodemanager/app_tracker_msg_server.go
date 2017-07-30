package nodemanager

import (
	"net"
	"sync"
	"time"

	"github.com/skycoin/skywire/messages"
)

type AppTrackerMsgServer struct {
	nm               *NodeManager
	conn             net.Conn
	maxPacketSize    int
	closeChannel     chan bool
	sequence         uint32
	responseChannels map[uint32]chan []byte
	timeout          time.Duration
	lock             *sync.Mutex
}

func newAppTrackerMsgServer(nm *NodeManager, serviceTrackerAddr string) (*AppTrackerMsgServer, error) {
	msgSrv := &AppTrackerMsgServer{}
	msgSrv.nm = nm

	msgSrv.responseChannels = make(map[uint32]chan []byte)

	msgSrv.maxPacketSize = config.MaxPacketSize
	msgSrv.timeout = time.Duration(config.MsgSrvTimeout) * time.Millisecond

	conn, err := net.Dial("tcp", serviceTrackerAddr)
	if err != nil {
		panic(err)
		return nil, err
	}

	msgSrv.conn = conn
	msgSrv.closeChannel = make(chan bool)

	msgSrv.lock = &sync.Mutex{}

	go msgSrv.receiveLoop()

	return msgSrv, nil
}

// close
func (self *AppTrackerMsgServer) shutdown() {
	self.closeChannel <- true
}

func (self *AppTrackerMsgServer) send(payload []byte) ([]byte, error) {

	responseChannel := make(chan []byte)
	sequence := self.sequence
	self.lock.Lock()
	self.responseChannels[sequence] = responseChannel
	self.lock.Unlock()
	self.sequence++

	msg := messages.ServiceRequest{
		payload,
		sequence,
	}

	msgS := messages.Serialize(messages.MsgServiceRequest, msg)
	_, err := self.conn.Write(msgS)
	if err != nil {
		return []byte{}, err
	}

	select {
	case response := <-responseChannel:
		self.lock.Lock()
		delete(self.responseChannels, sequence)
		self.lock.Unlock()
		return response, nil
	case <-time.After(self.timeout * time.Millisecond):
		return []byte{}, messages.ERR_MSG_SRV_TIMEOUT
	}
}

func (self *AppTrackerMsgServer) receiveLoop() {
	go_on := true
	go func() {
		for go_on {

			buffer := make([]byte, self.maxPacketSize)

			n, err := self.conn.Read(buffer)

			if err != nil {
				if !go_on && n == 0 {
					break
				} else {
					panic(err)
				}
			} else {
				go self.handleServiceMessage(buffer[:n])
			}
		}
	}()
	<-self.closeChannel
	go_on = false
	self.conn.Close()
}

func (self *AppTrackerMsgServer) getResponse(sequence uint32, response []byte) {
	self.lock.Lock()
	responseChannel, ok := self.responseChannels[sequence]
	self.lock.Unlock()
	if !ok {
		return
	}
	responseChannel <- response
}

func (self *AppTrackerMsgServer) handleServiceMessage(msgS []byte) {
	msg := &messages.ServiceResponse{}
	err := messages.Deserialize(msgS, msg)
	if err == nil {
		self.getResponse(msg.Sequence, msg.Payload)
	}
}
