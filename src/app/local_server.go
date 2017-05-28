package app

import (
	"io"
	"net"
	"sync"

	"github.com/skycoin/skywire/src/messages"
)

type Server struct {
	app
}

func NewServer(appId messages.AppId, nodeAddr string, handle func([]byte) []byte) (*Server, error) {

	server := &Server{}
	server.id = appId
	server.appType = "internal_server"
	server.lock = &sync.Mutex{}
	server.timeout = APP_TIMEOUT
	server.handle = handle
	server.responseNodeAppChannels = make(map[uint32]chan []byte)

	err := server.RegisterAtNode(nodeAddr)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (self *Server) consume(appMsg *messages.AppMessage) {

	sequence := appMsg.Sequence
	go func() {
		responsePayload := self.handle(appMsg.Payload)
		response := &messages.AppMessage{
			sequence,
			responsePayload,
		}
		responseSerialized := messages.Serialize(messages.MsgAppMessage, response)
		self.sendToMeshnet(responseSerialized)
	}()
}

func (self *Server) RegisterAtNode(nodeAddr string) error {

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
	if err != nil {
		return err
	}

	resp := &messages.AppRegistrationResponse{}
	err = messages.Deserialize(respS, resp)
	if err != nil || !resp.Ok {
		return err
	}

	return nil
}

func (self *Server) listenFromNode() {
	conn := self.nodeConn
	for {
		message, err := getFullMessage(conn)
		if err != nil {
			if err == io.EOF {
				continue
			} else {
				break
			}
		} else {
			go self.handleIncomingFromNode(message)
		}
	}
}

func (self *Server) handleIncomingFromNode(msg []byte) error {
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
