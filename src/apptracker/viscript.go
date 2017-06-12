package apptracker

import (
	"log"

	vsmsg "github.com/corpusc/viscript/msg"

	"github.com/skycoin/skywire/src/viscript"
)

type ATViscriptServer struct {
	viscript.ViscriptServer
	appTracker *AppTracker
}

func (self *AppTracker) TalkToViscript(sequence, appId uint32) {
	vs := &ATViscriptServer{appTracker: self}
	self.viscriptServer = vs
	vs.Init(sequence, appId)
}

func (self *ATViscriptServer) handleUserCommand(uc *vsmsg.MessageUserCommand) {
	log.Println("command received:", uc)
	sequence := uc.Sequence
	appId := uc.AppId
	message := uc.Payload

	switch vsmsg.GetType(message) {

	case vsmsg.TypePing:
		ack := &vsmsg.MessagePingAck{}
		ackS := vsmsg.Serialize(vsmsg.TypePingAck, ack)
		self.SendAck(ackS, sequence, appId)

	case vsmsg.TypeResourceUsage:
		cpu, memory, err := self.GetResources()
		if err == nil {
			ack := &vsmsg.MessageResourceUsageAck{
				cpu,
				memory,
			}
			ackS := vsmsg.Serialize(vsmsg.TypeResourceUsageAck, ack)
			self.SendAck(ackS, sequence, appId)
		}

	case vsmsg.TypeShutdown:
		self.appTracker.Shutdown()
		ack := &vsmsg.MessageShutdownAck{}
		ackS := vsmsg.Serialize(vsmsg.TypeShutdownAck, ack)
		self.SendAck(ackS, sequence, appId)
		panic("goodbye")

	default:
		log.Println("Unknown user command:", message)
	}
}
