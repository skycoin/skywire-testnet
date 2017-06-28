package node

import (
	"errors"
	"io"
	"log"
	"net"

	"github.com/skycoin/skywire/messages"
)

const PACKET_SIZE = 1024

func (self *Node) listenForApps() {
	listenAddress := "0.0.0.0:" + self.appTalkPort

	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		panic(err)
	}

	for {
		appConn, err := l.Accept() // create a connection with the user app (e.g. browser)
		if err != nil {
			log.Println("Cannot accept client's connection")
			return
		}
		defer appConn.Close()

		go func() { // run listening the connection for data and sending it through the meshnet to the server
			for {
				sizeMessage := make([]byte, 8)

				_, err := appConn.Read(sizeMessage)
				if err != nil {
					if err == io.EOF {
						continue
					} else {
						return
					}
				}

				size := messages.BytesToNum(sizeMessage)

				message := make([]byte, size)

				_, err = io.ReadFull(appConn, message)
				if err != nil {
					return
				}

				nodeAppMsg := &messages.NodeAppMessage{}
				err = messages.Deserialize(message, nodeAppMsg)
				if err == nil {
					self.handleNodeAppMessage(nodeAppMsg, appConn)
				}
			}
		}()
	}
}

func (self *Node) sendMessageToApp(appId messages.AppId, msg []byte) error {
	self.lock.Lock()
	sequence := self.appSequence
	self.appSequence++
	self.lock.Unlock()
	message := messages.NodeAppMessage{
		sequence,
		appId,
		msg,
	}
	messageS := messages.Serialize(messages.MsgNodeAppMessage, message)

	return self.sendCtrlToApp(appId, messageS)
}

func (self *Node) sendResponseToApp(sequence uint32, appId messages.AppId, misc []byte) error {
	resp := messages.NodeAppResponse{
		sequence,
		misc,
	}
	respS := messages.Serialize(messages.MsgNodeAppResponse, resp)

	return self.sendCtrlToApp(appId, respS)
}

func (self *Node) sendCtrlToApp(appId messages.AppId, msg []byte) error {
	appIdStr := string(appId)
	appConn, ok := self.appConns[appIdStr]
	if !ok {
		return messages.ERR_APP_DOESNT_EXIST
	}
	err := sendToAppConn(appConn, msg)
	return err
}

func sendToAppConn(appConn net.Conn, msg []byte) error {
	sizeMessage := messages.NumToBytes(len(msg), 8)

	_, err := appConn.Write(sizeMessage)
	if err != nil {
		return err
	}

	_, err = appConn.Write(msg)
	return err
}

func (self *Node) handleNodeAppMessage(msg *messages.NodeAppMessage, appConn net.Conn) {

	switch messages.GetMessageType(msg.Payload) {

	case messages.MsgRegisterAppMessage:
		go self.registerApp(msg, appConn)

	case messages.MsgAppListRequest:
		go self.getAppsList(msg)

	case messages.MsgSendFromAppMessage:
		go self.sendFromApp(msg)

	case messages.MsgConnectToAppMessage:
		go self.connectApps(msg)

	default:
		log.Println(messages.ERR_INCORRECT_MESSAGE_TYPE.Error(), msg)
	}
}

func (self *Node) registerApp(msg *messages.NodeAppMessage, appConn net.Conn) error {

	registerAppMsgS := msg.Payload
	registerAppMsg := &messages.RegisterAppMessage{}
	err := messages.Deserialize(registerAppMsgS, registerAppMsg)
	if err != nil {
		return err
	}

	appType := registerAppMsg.AppType

	appId := msg.AppId
	appIdStr := string(appId)

	self.lock.Lock()
	if _, ok := self.appConns[appIdStr]; !ok {
		self.appConns[appIdStr] = appConn
	}
	self.lock.Unlock()

	respS, err := self.sendRegisterAppToServer(appIdStr, appType)
	if err != nil {
		return err
	}

	resp := &messages.AppRegistrationResponse{}
	err = messages.Deserialize(respS, resp)
	if err != nil {
		return err
	}

	err = self.sendResponseToApp(msg.Sequence, appId, respS)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return errors.New(resp.Error)
	}
	return nil
}

func (self *Node) getAppsList(msg *messages.NodeAppMessage) error {

	respS, err := self.sendAppListRequestToServer(msg.Payload)
	if err != nil {
		return err
	}

	err = self.sendResponseToApp(msg.Sequence, msg.AppId, respS)
	return err
}

func (self *Node) sendFromApp(msg *messages.NodeAppMessage) error {
	sfaS := msg.Payload
	sfa := &messages.SendFromAppMessage{}

	err := messages.Deserialize(sfaS, sfa)
	if err != nil {
		return err
	}

	self.lock.Lock()
	meshConn, ok := self.connections[sfa.ConnectionId]
	if !ok {
		return messages.ERR_CONNECTION_DOESNT_EXIST
	}
	self.lock.Unlock()

	meshConn.Send(sfa.Payload)
	err = self.sendResponseToApp(msg.Sequence, msg.AppId, []byte{})
	return err
}

func (self *Node) connectApps(msg *messages.NodeAppMessage) error {
	connectMsgS := msg.Payload
	connectMsg := &messages.ConnectToAppMessage{}

	err := messages.Deserialize(connectMsgS, connectMsg)
	if err != nil {
		return err
	}

	_, err = self.Dial(connectMsg.Address, connectMsg.AppFrom, connectMsg.AppTo)
	if err != nil {
		return err
	}

	err = self.sendResponseToApp(msg.Sequence, msg.AppId, []byte{})
	return err
}
