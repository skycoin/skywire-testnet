package hypervisor

import (
	"errors"
	"strconv"

	"github.com/skycoin/viscript/msg"
)

var ExtProcessListGlobal ExtProcessList

type ExtProcessList struct {
	ProcessMap map[msg.ExtProcessId]msg.ExtProcessInterface
}

func initExtProcessList() {
	ExtProcessListGlobal.ProcessMap = make(map[msg.ExtProcessId]msg.ExtProcessInterface)
}

func teardownExtProcessList() {
	ExtProcessListGlobal.ProcessMap = nil
	// TODO: Further cleanup
}

func ExtProcessIsRunning(procId msg.ExtProcessId) bool {
	_, exists := ExtProcessListGlobal.ProcessMap[procId]
	return exists
}

func AddExtProcess(ep msg.ExtProcessInterface) msg.ExtProcessId {
	id := ep.GetId()

	if !ExtProcessIsRunning(id) {
		ExtProcessListGlobal.ProcessMap[id] = ep
	}

	return id
}

func GetExtProcess(id msg.ExtProcessId) (msg.ExtProcessInterface, error) {
	extProc, exists := ExtProcessListGlobal.ProcessMap[id]
	if exists {
		return extProc, nil
	}

	err := errors.New("External process with id " +
		strconv.Itoa(int(id)) + " doesn't exist!")

	return nil, err
}

func RemoveExtProcess(id msg.ExtProcessId) {
	delete(ExtProcessListGlobal.ProcessMap, id)
}

func TickExtTasks() {
	// TODO: Read from response channels if they contain any new messages
	// for _, p := range ExtProcessListGlobal.ProcessMap {
	// data, err := monitor.Monitor.ReadFrom(p.GetId())
	// if err != nil {
	// 	// println(err.Error())
	// 	// monitor.Monitor.PrintAll()
	// 	continue
	// }

	// ackType := msg.GetType(data)

	// switch ackType {
	// case msg.TypeUserCommandAck:

	// }

	// select {
	// case <-p.GetProcessExitChannel():
	// 	println("Got the exit in task ext list")
	// default:
	// }
	// p.Tick()
	// }

}
