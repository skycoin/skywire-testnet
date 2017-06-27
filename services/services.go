package services

import (
	"log"
	"net"
)

type Service struct {
	conn          net.Conn
	maxPacketSize int
	closeChannel  chan bool
}

func (self *Service) Init(nmAddr, serviceId uint32) {
	conn, err := net.Dial("tcp", nmAddr)
	if err != nil {
		panic(err)
	}
	self.conn = conn
	log.Println("Waiting for orchestration server messages")
	self.sendFirstAck(sequence, serviceId)
	self.serve()
}

func (self *Service) Shutdown() {
	self.conn.Close()
}

func (self *Service) serve() {
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
				if n > 0 {
					log.Printf("connection at %s received %d bytes\n", self.conn.LocalAddr().String(), n)
					uc := vsmsg.MessageUserCommand{}
					err := vsmsg.Deserialize(buffer[:n], &uc)
					if err != nil {
						log.Println("Incorrect UserCommand:", buffer[:n])
						continue
					}
					go self.handleUserCommand(&uc)
				}
			}
		}
	}()
	<-self.closeChannel
	go_on = false
	self.conn.Close()
}

func (self *Service) handleUserCommand(uc *vsmsg.MessageUserCommand) {
	log.Println("command received:", uc)
}

func (self *Service) SendAck(ackS []byte, sequence, serviceId uint32) {
	ucAck := &vsmsg.MessageUserCommandAck{
		sequence,
		serviceId,
		ackS,
	}
	ucAckS := vsmsg.Serialize(vsmsg.TypeUserCommandAck, ucAck)
	self.send(ucAckS)
}

func (self *Service) send(data []byte) {
	_, err := self.conn.Write(data)
	if err != nil {
		log.Println("Unsuccessful sending to services from nodemanager")
	}
}

func (self *Service) sendFirstAck(sequence, serviceId uint32) {
	ack := vsmsg.MessageCreateAck{}
	ackS := vsmsg.Serialize(vsmsg.TypeCreateAck, ack)
	self.SendAck(ackS, sequence, serviceId)
}
