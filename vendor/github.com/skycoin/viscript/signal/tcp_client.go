package signal

import (
	"net"
	"io"
	"log"
	sgmsg "github.com/skycoin/viscript/signal/msg"
	"runtime"
	"time"
	"io/ioutil"
	"strings"
	"strconv"
	"fmt"
)

const viscriptAddr = "127.0.0.1:7999"

type SignalNode struct {
	port          string
	appId         uint32
	serverAddress string
}

func InitSignalNode(port string) *SignalNode {
	client := &SignalNode{port: port,
		appId: 1,
		serverAddress: viscriptAddr,
	}
	return client
}

func (self *SignalNode) ListenForSignals() {
	listenAddress := "0.0.0.0:" + self.port
	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		panic(err)
	}
	//Send first message to signal-server
	ack := &sgmsg.MessageFirstConnect{Address: "127.0.0.1", Port: self.port}
	ackS := sgmsg.Serialize(sgmsg.TypeFirstConnect, ack)
	self.SendAck(ackS, 1, self.appId)
	log.Println("Listen for incoming message on port: " + self.port)
	for {
		appConn, err := l.Accept() // create a connection with the user app (e.g. browser)
		if err != nil {
			log.Println("Cannot accept client's connection")
			return
		}
		defer appConn.Close()

		go func() { // run listening the connection for data and sending it through the meshnet to the server
			for {
				sizeMessage := make([]byte, 56)
				_, err := appConn.Read(sizeMessage)
				if err != nil {
					if err == io.EOF {
						continue
					} else {
						log.Println(err)
					}

				}


				switch sgmsg.GetType(sizeMessage) {

				case sgmsg.TypeUserCommand:
					uc := sgmsg.MessageUserCommand{}
					err = sgmsg.Deserialize(sizeMessage, &uc)
					if err != nil {
						log.Println("Incorrect UserCommand:", sizeMessage)
						continue
					}


					self.handleUserCommand(&uc)

				default:
					log.Println("Bad command")
				}
			}
		}()
	}
}

func (self *SignalNode) handleUserCommand(uc *sgmsg.MessageUserCommand) {
	sequence := uc.Sequence
	//appId := uc.AppId
	message := uc.Payload

	test := sgmsg.MessageUserCommand{}
	err := sgmsg.Deserialize(uc.Payload, &test)
	if err != nil {
		log.Println("Incorrect UserCommand:", uc.Payload)
	}

	switch sgmsg.GetType(test.Payload) {

	case sgmsg.TypePing:
		log.Println("ping command received")
		ack := &sgmsg.MessagePingAck{}
		ackS := sgmsg.Serialize(sgmsg.TypePingAck, ack)
		self.SendAck(ackS, sequence, self.appId)

	case sgmsg.TypeResourceUsage:
		log.Println("res_usage command received")
		cpu, memory, err := GetResources()
		if err == nil {
			ack := &sgmsg.MessageResourceUsageAck{
				cpu,
				memory,
			}
			ackS := sgmsg.Serialize(sgmsg.TypeResourceUsageAck, ack)
			self.SendAck(ackS, sequence, self.appId)
		}

	case sgmsg.TypeShutdown:
		log.Println("shutdown command received")
		shutdown := sgmsg.MessageShutdown{}
		err = sgmsg.Deserialize(test.Payload, &shutdown)
		if err != nil {
			panic(err)
		}

		switch shutdown.Stage {
			case 1:
				log.Println("app is preparing for shutdown... ")
				ack := &sgmsg.MessageShutdownAck{Stage: 1}
				ackS := sgmsg.Serialize(sgmsg.TypeShutdownAck, ack)
				self.SendAck(ackS, sequence, self.appId)
			case 2:
				log.Println("turn off daemons... ")
				self.TurnOffNodes()
				ack := &sgmsg.MessageShutdownAck{Stage: 2}
				ackS := sgmsg.Serialize(sgmsg.TypeShutdownAck, ack)
				self.SendAck(ackS, sequence, self.appId)
			case 3:
				ack := &sgmsg.MessageShutdownAck{Stage: 3}
				ackS := sgmsg.Serialize(sgmsg.TypeShutdownAck, ack)
				self.SendAck(ackS, sequence, self.appId)
				panic("goodbye")
		}

	case sgmsg.TypeStartup:
		startup := sgmsg.MessageStartup{}
		err = sgmsg.Deserialize(test.Payload, &startup)
		if err != nil {
			panic(err)
		}

		switch startup.Stage {
		case 1:
			self.serverAddress = startup.Address
			log.Println("app is preparing for shutdown... ")
			ack := &sgmsg.MessageStartupAck{Stage: 1}
			ackS := sgmsg.Serialize(sgmsg.TypeStartupAck, ack)
			self.SendAck(ackS, sequence, self.appId)
		case 2:
			log.Println("turn on daemons... ")
			self.TurnOnNodes()
			ack := &sgmsg.MessageStartupAck{Stage: 2}
			ackS := sgmsg.Serialize(sgmsg.TypeStartupAck, ack)
			self.SendAck(ackS, sequence, self.appId)
		case 3:
			ack := &sgmsg.MessageStartupAck{Stage: 3}
			ackS := sgmsg.Serialize(sgmsg.TypeStartupAck, ack)
			self.SendAck(ackS, sequence, self.appId)
			log.Println("signal-server is connected.")
		}


	default:
		log.Println("Unknown user command:", message)

	}
}


//This is empty func, need to add functionality
func (self *SignalNode) TurnOffNodes(){
	time.Sleep(1* time.Second)
	log.Println("Daemons turned off")
}

//This is empty func, need to add functionality
func (self *SignalNode) TurnOnNodes(){
	time.Sleep(1* time.Second)
	log.Println("Daemons turned on")
}

func (self *SignalNode) SendAck(ackS []byte, sequence, appId uint32) {
	ucAck := &sgmsg.MessageUserCommandAck{
		sequence,
		self.appId,
		ackS,
	}
	ucAckS := sgmsg.Serialize(sgmsg.TypeUserCommandAck, ucAck)
	self.send(ucAckS)
}

func (self *SignalNode) send(data []byte) {
	conn, e := net.Dial("tcp", self.serverAddress)
	if e != nil {
		log.Println("bad conn")
	}
	_, err := conn.Write(data)
	if err != nil {
		log.Println("Unsuccessful sending")
	}
}

//Need realization for macOS and windows
func GetResources() (float64, uint64, error) {
	var cpu float64
	switch runtime.GOOS {
	case "linux":
		cpu = CPUUsage()
	case "darwin":
		log.Println("darwin")
	case "windows":
		log.Println("windows")
	default:
		log.Println("unknow OS")
	}
	return cpu, getMemStats(), nil
}

func getMemStats() uint64 {
	ms := &runtime.MemStats{}
	runtime.ReadMemStats(ms)
	return ms.Alloc
}

func getCPUSample() (idle, total uint64) {
	contents, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range(lines) {
		fields := strings.Fields(line)
		if fields[0] == "cpu" {
			numFields := len(fields)
			for i := 1; i < numFields; i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					fmt.Println("Error: ", i, fields[i], err)
				}
				total += val // tally up all the numbers to get total ticks
				if i == 4 {  // idle is the 5th field in the cpu line
					idle = val
				}
			}
			return
		}
	}
	return
}

func CPUUsage() float64{
	idle0, total0 := getCPUSample()
	time.Sleep(1 * time.Second)
	idle1, total1 := getCPUSample()

	idleTicks := float64(idle1 - idle0)
	totalTicks := float64(total1 - total0)
	cpuUsage := 100 * (totalTicks - idleTicks) / totalTicks
	return cpuUsage
}