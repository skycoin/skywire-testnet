package nodemanager

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skywire/messages"
	"fmt"
)

type NodeMsgServer struct {
	nm               *NodeManager
	factory          *factory.MessengerFactory
	maxPacketSize    int
	nodeAddrs        map[cipher.PubKey]*net.UDPAddr
	sequence         uint32
	responseChannels map[uint32]chan bool
	timeout          time.Duration
	lock             *sync.Mutex
}

func newNodeMsgServer(nm *NodeManager) (*NodeMsgServer, error) {
	msgSrv := &NodeMsgServer{}
	msgSrv.nm = nm

	msgSrv.responseChannels = make(map[uint32]chan bool)

	msgSrv.maxPacketSize = config.MaxPacketSize
	msgSrv.timeout = time.Duration(config.MsgSrvTimeout) * time.Millisecond

	factory := factory.NewMessengerFactory()
	factory.CustomMsgHandler = msgSrv.msgHandler
	go func() {
		err := factory.Listen(nm.ctrlAddr)
		if err != nil {
			log.Printf("messenger factory listen failed %v", err)
		}
	}()

	msgSrv.factory = factory

	msgSrv.nodeAddrs = make(map[cipher.PubKey]*net.UDPAddr)

	msgSrv.lock = &sync.Mutex{}
	msgSrv.sequence = uint32(1) // 0 for no-wait sends

	return msgSrv, nil
}

// close
func (self *NodeMsgServer) shutdown() {
	self.factory.Close()
}

func (self *NodeMsgServer) sendMessage(node cipher.PubKey, msg []byte) error {

	responseChannel := make(chan bool)
	sequence := self.sequence
	self.lock.Lock()
	self.responseChannels[sequence] = responseChannel
	self.lock.Unlock()
	self.sequence++

	err := self.send(sequence, node, msg)
	if err != nil {
		return err
	}

	select {
	case <-responseChannel:
		return nil
	case <-time.After(self.timeout * time.Millisecond):
		return messages.ERR_MSG_SRV_TIMEOUT
	}
}

func (self *NodeMsgServer) sendNoWait(node cipher.PubKey, msg []byte) error {
	err := self.send(uint32(0), node, msg)
	return err
}

func (self *NodeMsgServer) sendAck(sequence uint32, node cipher.PubKey, msg []byte) error {
	err := self.send(sequence, node, msg)
	return err
}

func (self *NodeMsgServer) send(sequence uint32, node cipher.PubKey, msg []byte) error {
	inControlMsg := messages.InControlMessage{
		messages.ChannelId(0),
		sequence,
		msg,
	}
	inControlS := messages.Serialize(messages.MsgInControlMessage, inControlMsg)
	log.Printf("inControlS %x", inControlS)
	conn, ok := self.factory.GetConnection(node)
	if !ok {
		return fmt.Errorf("node %s not found", node.Hex())
	}
	return conn.SendCustom(inControlS)
}

func (self *NodeMsgServer) msgHandler(conn *factory.Connection, msg []byte) {
	cm := messages.InControlMessage{}
	err := messages.Deserialize(msg, &cm)
	if err != nil {
		log.Println("Incorrect InControlMessage:", msg)
	}

	go self.nm.handleControlMessage(conn, &cm)
}

func (self *NodeMsgServer) getResponse(sequence uint32, response *messages.CommonCMAck) {
	self.lock.Lock()
	responseChannel, ok := self.responseChannels[sequence]
	self.lock.Unlock()
	if !ok {
		return
	}
	responseChannel <- response.Ok
}
