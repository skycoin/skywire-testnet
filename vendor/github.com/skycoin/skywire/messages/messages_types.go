package messages

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	MsgInRouteMessage            = iota // Transport -> Node
	MsgOutRouteMessage                  // Node -> Transport
	MsgTransportDatagramTransfer        // Transport -> Transport, simulating sending packet over network
	MsgTransportDatagramACK             // Transport -> Transport, simulating ACK for packet
	MsgCongestionPacket                 // Transport -> Transport

	MsgConnectionMessage // Connection -> Connection
	MsgConnectionAck     // Connection -> Connection

	MsgProxyMessage // Application -> Application
	MsgAppMessage   // Application -> Application

	MsgInControlMessage           // Transport -> Node, control message
	MsgOutControlMessage          // Node -> Transport, control message
	MsgCloseChannelControlMessage // Node -> Control channel, close control channel
	MsgAddRouteCM                 // Node -> Control channel, add new route
	MsgRemoveRouteCM              // Node -> Control channel, remove route
	MsgRegisterNodeCM             // Node -> NodeManager
	MsgRegisterNodeCMAck          // NodeManager -> Node
	MsgAssignPortCM               // NodeManager -> Node
	MsgTransportCreateCM          // NodeManager -> Node
	MsgTransportTickCM            // NodeManager -> Node
	MsgTransportShutdownCM        // NodeManager -> Node
	MsgOpenUDPCM                  // NodeManager -> Node
	MsgCommonCMAck                // Node -> NodeManager, NodeManager -> Node
	MsgConnectDirectlyCM          // Node -> NodeManager
	MsgConnectDirectlyCMAck       // NodeManager -> Node
	MsgConnectWithRouteCM         // Node -> NodeManager
	MsgConnectWithRouteCMAck      // NodeManager -> Node
	MsgAssignConnectionCM         // NodeManager -> Node
	MsgConnectionOnCM             // NodeManager -> Node
	MsgRegisterAppCM              // Node -> NodeManager
	MsgRegisterAppCMAck           // NodeManager -> Node
	MsgAppListRequestCM           // Node -> NodeManager
	MsgShutdownCM                 // NodeManager -> Node

	MsgNodeAppMessage      // Application -> Node
	MsgNodeAppResponse     // Node -> Application
	MsgSendFromAppMessage  // Application -> Node
	MsgRegisterAppMessage  // Application -> Node
	MsgConnectToAppMessage // Application -> Node
	MsgAssignConnectionNAM // Node -> Application

	MsgServiceRequest          // NodeManager -> Service
	MsgServiceResponse         // Service -> NodeManager
	MsgAppRegistrationRequest  // NodeManager -> AppTracker
	MsgAppRegistrationResponse // AppTracker -> NodeManager
	MsgAppListRequest          // NodeManager -> AppTracker
	MsgAppListResponse         // AppTracker -> NodeManager
)

func GetMessageType(message []byte) uint16 {
	var value uint16
	rBuf := bytes.NewReader(message[0:2])
	err := binary.Read(rBuf, binary.LittleEndian, &value)
	if err != nil {
		fmt.Println("binary.Read failed: ", err)
	}
	return value
}
