package node

import (
	"errors"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/skycoin/skywire/messages"
)

func (self *Node) addControlChannel() messages.ChannelId {

	channel := newControlChannel()

	self.lock.Lock()
	defer self.lock.Unlock()

	self.controlChannels[channel.id] = channel
	return channel.id
}

func (self *Node) addZeroControlChannel() {

	channel := newControlChannel()
	channel.id = messages.ChannelId(0)

	self.lock.Lock()
	defer self.lock.Unlock()

	self.controlChannels[channel.id] = channel
	return
}

func (self *Node) closeControlChannel(channelID messages.ChannelId) error {

	if _, ok := self.controlChannels[channelID]; !ok {
		return errors.New("Control channel not found")
	}

	self.lock.Lock()
	defer self.lock.Unlock()

	delete(self.controlChannels, channelID)
	return nil
}

func (self *Node) handleControlMessage(_ messages.ChannelId, inControlMsg *messages.InControlMessage) error {

	channelID := messages.ChannelId(0)

	self.lock.Lock()
	channel, ok := self.controlChannels[channelID]
	self.lock.Unlock()
	if !ok {
		return errors.New("Control channel not found")
	}

	sequence := inControlMsg.Sequence
	msg := inControlMsg.PayloadMessage
	err := channel.handleMessage(self, sequence, msg)
	return err
}

func (self *Node) sendTrueAckToServer(sequence uint32) error {
	ack := &messages.CommonCMAck{true}
	return self.sendAckToServer(sequence, ack)
}

func (self *Node) sendFalseAckToServer(sequence uint32) error {
	ack := &messages.CommonCMAck{false}
	return self.sendAckToServer(sequence, ack)
}

func (self *Node) sendAckToServer(sequence uint32, ack *messages.CommonCMAck) error {
	ackS := messages.Serialize(messages.MsgCommonCMAck, *ack)
	return self.sendToServer(sequence, ackS)
}

func (self *Node) sendRegisterNodeToServer(hostname, host string, connect bool) error {
	msg := messages.RegisterNodeCM{hostname, host, connect}
	msgS := messages.Serialize(messages.MsgRegisterNodeCM, msg)
	_, err := self.sendMessageToServer(msgS)
	return err
}

func (self *Node) sendRegisterAppToServer(appId, appType string) ([]byte, error) {
	var host string
	if self.hostname == "" {
		host = self.id.Hex()
	} else {
		host = self.hostname
	}
	msg := messages.RegisterAppCM{messages.ServiceInfo{appId, appType, host}, self.id}
	msgS := messages.Serialize(messages.MsgRegisterAppCM, msg)
	result, err := self.sendMessageToServer(msgS)
	return result, err
}

func (self *Node) sendAppListRequestToServer(request []byte) ([]byte, error) {
	msg := messages.AppListRequestCM{request, self.id}
	msgS := messages.Serialize(messages.MsgAppListRequestCM, msg)
	result, err := self.sendMessageToServer(msgS)
	return result, err
}

func (self *Node) sendConnectDirectlyToServer(nodeToId string) error {
	responseChannel := make(chan bool)

	self.lock.Lock()
	connectSequence := self.connectResponseSequence
	self.connectResponseSequence++
	self.connectResponseChannels[connectSequence] = responseChannel
	self.lock.Unlock()

	msg := messages.ConnectDirectlyCM{connectSequence, self.id, nodeToId}
	msgS := messages.Serialize(messages.MsgConnectDirectlyCM, msg)

	_, err := self.sendMessageToServer(msgS)
	if err != nil {
		return err
	}

	select {
	case <-responseChannel:
		return nil
	case <-time.After(CONTROL_TIMEOUT):
		return messages.ERR_MSG_SRV_TIMEOUT
	}
}

func (self *Node) sendConnectWithRouteToServer(nodeToId string, appIdFrom, appIdTo messages.AppId) (messages.ConnectionId, error) {
	responseChannel := make(chan messages.ConnectionId)

	self.lock.Lock()
	connSequence := self.connectionResponseSequence
	self.connectionResponseSequence++
	self.connectionResponseChannels[connSequence] = responseChannel
	self.lock.Unlock()

	msg := messages.ConnectWithRouteCM{connSequence, appIdFrom, appIdTo, self.id, nodeToId}
	msgS := messages.Serialize(messages.MsgConnectWithRouteCM, msg)

	_, err := self.sendMessageToServer(msgS)
	if err != nil {
		return messages.ConnectionId(0), err
	}

	select {
	case connId := <-responseChannel:
		return connId, nil
	case <-time.After(CONTROL_TIMEOUT):
		return messages.ConnectionId(0), messages.ERR_MSG_SRV_TIMEOUT
	}
}

func (self *Node) sendMessageToServer(msg []byte) ([]byte, error) {
	self.lock.Lock()
	sequence := self.sequence
	self.sequence++
	self.lock.Unlock()

	responseChannel := make(chan []byte)
	self.setResponseChannel(sequence, responseChannel)

	err := self.sendToServer(sequence, msg)
	if err != nil {
		return []byte{}, err
	}

	select {
	case response := <-responseChannel:
		return response, nil
	case <-time.After(CONTROL_TIMEOUT):
		return []byte{}, messages.ERR_MSG_SRV_TIMEOUT
	}
}

func (self *Node) sendToServer(sequence uint32, msg []byte) error {
	if len(self.serverAddrs) == 0 {
		return nil
	}

	inControl := messages.InControlMessage{
		messages.ChannelId(0),
		sequence,
		msg,
	}
	inControlS := messages.Serialize(messages.MsgInControlMessage, inControl)
	return self.controlConn.SendCustom(inControlS)
}

func (self *Node) openUDPforCM(port int) (*net.UDPConn, error) {
	host := net.ParseIP(messages.LOCALHOST)
	connAddr := &net.UDPAddr{IP: host, Port: port}

	conn, err := net.ListenUDP("udp", connAddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (self *Node) addServer(serverAddrStr string) {

	nmData := strings.Split(serverAddrStr, ":")
	nmHostStr := nmData[0]
	nmPort := 5999
	if len(nmData) > 1 {
		port, err := strconv.Atoi(nmData[1])
		if err == nil {
			nmPort = port
		}
	}
	nmHost := net.ParseIP(nmHostStr)
	serverAddr := &net.UDPAddr{IP: nmHost, Port: nmPort}
	self.serverAddrs = append(self.serverAddrs, serverAddr)
}

func (self *Node) receiveControlMessages() {
	for {
		select {
		case m, ok := <-self.controlConn.GetChanIn():
			if !ok {
				return
			}
			if m[0] != 2 {
				continue
			}
			m = m[1:]
			log.Printf("InControlMessage:%x\n", m)
			cm := messages.InControlMessage{}
			err := messages.Deserialize(m, &cm)
			if err != nil {
				log.Printf("Incorrect InControlMessage:%x err:%v\n", m, err)
				continue
			}
			go self.injectControlMessage(&cm)
		}
	}
}
