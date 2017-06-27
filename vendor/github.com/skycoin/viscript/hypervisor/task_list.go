package hypervisor

import (
	"github.com/skycoin/viscript/msg"
)

/*
	Process can
	- receive messages from hypervisor
	- can emit message back to hypervisor
	- have a tick method for handling incoming messages

	Incoming messages from process to hypervisor come in anytime
	- on input dispatch
	Messages outgoing to hypervisor are processed in DispatchProcessEvents()
	- we check each process channel for outgoing messages to the hypervisor
	Each process has a tick() method
	- tick method, input messages are processed, output messages created
*/

var ProcessListGlobal ProcessList

type ProcessList struct {
	ProcessMap map[msg.ProcessId]msg.ProcessInterface //process id to interface
}

func initProcessList() {
	ProcessListGlobal.ProcessMap = make(map[msg.ProcessId]msg.ProcessInterface)
}

func teardownProcessList() {
	ProcessListGlobal.ProcessMap = nil
	// TODO: actually call teardown methods on all the processes and also
	// external processes. what about Alt+f4?
	// upon application exit we need to terminate all the running processes
	// and external processes
}

func AddProcess(p msg.ProcessInterface) msg.ProcessId {
	id := p.GetId()

	_, isInTheMap := ProcessListGlobal.ProcessMap[id]
	if !isInTheMap {
		ProcessListGlobal.ProcessMap[id] = p
	}
	return id
}

func GetProcessEvents() {
	println("process_list.GetProcessEvents()   ---------------- TODO !!!!!!!!!!!")
}

func TickTasks() {
	for _, p := range ProcessListGlobal.ProcessMap {
		p.Tick()
	}
}
