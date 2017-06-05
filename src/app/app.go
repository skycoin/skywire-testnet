package app

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/skycoin/skywire/src/messages"
)

type app struct {
	ProxyAddress            string
	id                      messages.AppId
	appType                 string
	nodeConn                net.Conn
	handle                  func([]byte) []byte
	timeout                 time.Duration
	meshConnId              messages.ConnectionId
	nodeAppSequence         uint32
	responseNodeAppChannels map[uint32]chan []byte
	lock                    *sync.Mutex
	viscriptServer          *AppViscriptServer
}

const PACKET_SIZE = 1024

var APP_TIMEOUT = 10000 * time.Duration(time.Millisecond)

func (self *app) Id() messages.AppId {
	return self.id
}

func (self *app) Connect(appId messages.AppId, address string) error {

	msg := messages.ConnectToAppMessage{
		address,
		self.id,
		appId,
	}

	msgS := messages.Serialize(messages.MsgConnectToAppMessage, msg)

	_, err := self.sendToNode(msgS)

	return err
}

func (self *app) Shutdown() {
	self.nodeConn.Close()
}

func (self *app) consume(_ *messages.AppMessage) {
	panic("STUB-CONSUMING, no consume method in app implementation")
	//stub
}

func (self *app) sendToMeshnet(payload []byte) error {

	msg := messages.SendFromAppMessage{
		self.meshConnId,
		payload,
	}

	msgS := messages.Serialize(messages.MsgSendFromAppMessage, msg)

	_, err := self.sendToNode(msgS)
	return err
}

func (self *app) sendToNode(payload []byte) ([]byte, error) {
	if self.nodeConn == nil {
		return []byte{}, nil // return error
	}

	respChan := make(chan []byte)

	sequence := self.setResponseNodeAppChannel(respChan)
	nodeAppMessage := messages.NodeAppMessage{
		sequence,
		self.id,
		payload,
	}

	msgS := messages.Serialize(messages.MsgNodeAppMessage, nodeAppMessage)
	sizeMessage := messages.NumToBytes(len(msgS), 8)

	_, err := self.nodeConn.Write(sizeMessage)
	if err != nil {
		return []byte{}, err
	}

	_, err = self.nodeConn.Write(msgS)
	if err != nil {
		return []byte{}, err
	}

	select {
	case response := <-respChan:
		return response, nil
	case <-time.After(self.timeout * time.Millisecond):
		return []byte{}, messages.ERR_APP_TIMEOUT
	}
}

func (self *app) getResponseNodeAppChannel(sequence uint32) (chan []byte, error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	ch, ok := self.responseNodeAppChannels[sequence]
	if !ok {
		return nil, messages.ERR_NO_APP_RESPONSE_CHANNEL
	}
	return ch, nil
}

func (self *app) setResponseNodeAppChannel(responseChannel chan []byte) uint32 {
	self.lock.Lock()
	defer self.lock.Unlock()

	sequence := self.nodeAppSequence
	self.nodeAppSequence++
	self.responseNodeAppChannels[sequence] = responseChannel
	return sequence
}

func (self *app) RegisterAtNode(nodeAddr string) error {

	nodeConn, err := net.Dial("tcp", nodeAddr)
	if err != nil {
		panic(err)
		return err
	}

	self.nodeConn = nodeConn

	go self.listenFromNode()

	registerMessage := messages.RegisterAppMessage{
		self.appType,
	}

	rmS := messages.Serialize(messages.MsgRegisterAppMessage, registerMessage)

	respS, err := self.sendToNode(rmS)
	resp := &messages.AppRegistrationResponse{}

	err = messages.Deserialize(respS, resp)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return errors.New(resp.Error)
	}

	return nil
}

func (self *app) GetAppsList(requestType, requestParam string) ([]messages.ServiceInfo, error) {
	request := messages.AppListRequest{requestType, requestParam}
	requestS := messages.Serialize(messages.MsgAppListRequest, request)

	respS, err := self.sendToNode(requestS)
	if err != nil {
		return nil, err
	}

	resp := &messages.AppListResponse{}
	err = messages.Deserialize(respS, resp)
	if err != nil {
		return nil, err
	}

	return resp.Apps, nil
}

func (self *app) listenFromNode() {
	conn := self.nodeConn
	for {
		message := make([]byte, PACKET_SIZE)

		n, err := conn.Read(message)
		if err != nil {
			return
			if err == io.EOF {
				continue
			} else {
				break
			}
		}

		self.handleIncomingFromNode(message[:n])
	}
}

func (self *app) handleIncomingFromNode(msg []byte) error {
	switch messages.GetMessageType(msg) {

	case messages.MsgAssignConnectionNAM:
		m1 := &messages.AssignConnectionNAM{}
		err := messages.Deserialize(msg, m1)
		if err != nil {
			return err
		}
		self.meshConnId = m1.ConnectionId
		return nil

	case messages.MsgAppMessage:
		appMsg := &messages.AppMessage{}
		err := messages.Deserialize(msg, appMsg)
		if err != nil {
			return err
		}
		go self.consume(appMsg)
		return nil

	case messages.MsgNodeAppResponse:
		nar := &messages.NodeAppResponse{}
		err := messages.Deserialize(msg, nar)
		if err != nil {
			return err
		}

		sequence := nar.Sequence
		respChan, err := self.getResponseNodeAppChannel(sequence)
		if err != nil {
			panic(err)
			return err
		} else {
			respChan <- nar.Misc
			return nil
		}

	default:
		return messages.ERR_INCORRECT_MESSAGE_TYPE
	}
}
