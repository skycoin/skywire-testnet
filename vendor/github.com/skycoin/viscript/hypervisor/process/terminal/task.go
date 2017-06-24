package process

import (
	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/hypervisor"
	"github.com/skycoin/viscript/msg"
)

var path = "hypervisor/process/terminal/task"

type Process struct {
	Id           msg.ProcessId
	Type         msg.ProcessType
	Label        string
	OutChannelId uint32
	InChannel    chan []byte
	State        State

	hasExtProcAttached bool
	attachedExtProcess msg.ExtProcessInterface
}

//non-instanced
func MakeNewTask() *Process {
	println("<" + path + ">.MakeNewTask()")

	var p Process
	p.Id = msg.NextProcessId()
	p.Type = 0
	p.Label = "TestLabel"
	p.InChannel = make(chan []byte, msg.ChannelCapacity)
	p.State.Init(&p)

	//means no external task is attached
	p.hasExtProcAttached = false

	return &p
}

func (pr *Process) GetProcessInterface() msg.ProcessInterface {
	app.At(path, "GetProcessInterface")
	return msg.ProcessInterface(pr)
}

func (pr *Process) DeleteProcess() {
	app.At(path, "DeleteProcess")
	close(pr.InChannel)
	pr.State.proc = nil
	pr = nil
}

func (pr *Process) HasExtProcessAttached() bool {
	return pr.hasExtProcAttached
}

func (pr *Process) AttachExternalProcess(extProc msg.ExtProcessInterface) error {
	app.At(path, "AttachExternalProcess")
	err := extProc.Attach()
	if err != nil {
		return err
	}

	pr.attachedExtProcess = extProc
	pr.hasExtProcAttached = true

	return nil
}

func (pr *Process) DetachExternalProcess() {
	app.At(path, "DetachExternalProcess")
	// pr.attachedExtProcess.Detach()
	pr.attachedExtProcess = nil
	pr.hasExtProcAttached = false
}

func (pr *Process) ExitExtProcess() {
	app.At(path, "ExitExtProcess")

	//set flag false
	pr.hasExtProcAttached = false

	//store the exteral process id for removing from the global list
	extProcId := pr.attachedExtProcess.GetId()

	//teardown and cleanup external process
	pr.attachedExtProcess.TearDown()

	//set current attachedExtProcess to nil
	pr.attachedExtProcess = nil

	//remove from the ExtProcessListGlobal.ProcessMap
	hypervisor.RemoveExtProcess(extProcId)
}

//implement the interface

func (pr *Process) GetId() msg.ProcessId {
	return pr.Id
}

func (pr *Process) GetType() msg.ProcessType {
	return pr.Type
}

func (pr *Process) GetLabel() string {
	return pr.Label
}

func (pr *Process) GetIncomingChannel() chan []byte {
	return pr.InChannel
}

func (pr *Process) Tick() {
	pr.State.HandleMessages()

	if !pr.HasExtProcessAttached() {
		return
	}

	select {
	//case exit := <-pr.attachedExtProcess.GetProcessExitChannel():
	// if exit {
	// 	println("Got the exit in task, process is finished.")
	// 	//TODO: still not working yet. looking for the best way to finish
	// 	//multiple goroutines at the same time to avoid any side effects
	// 	pr.ExitExtProcess()
	// }
	case data := <-pr.attachedExtProcess.GetProcessOutChannel():
		println("Received data from external process, sending to term.")
		pr.State.PrintLn(string(data))
	default:
	}
}
