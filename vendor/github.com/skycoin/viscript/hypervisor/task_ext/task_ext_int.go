package task_ext

import (
	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/msg"
)

//ExtProcessInterface implementation

func (pr *ExternalProcess) Tick() {
	pr.processInput()
	pr.processOutput()
}

func (pr *ExternalProcess) Start() error {
	app.At(te, "Start")

	err := pr.cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func (pr *ExternalProcess) TearDown() {
	app.At(te, "TearDown")

	pr.cmd.Process.Kill()

	close(pr.cmdIn)
	close(pr.cmdOut)

	close(pr.ProcessIn)
	close(pr.ProcessOut)
	// close(pr.ProcessExit)

	if pr.cmd != nil {
		pr.cmd = nil
	}

	if pr.stdOutPipe != nil {
		pr.stdOutPipe = nil
	}

	if pr.stdInPipe != nil {
		pr.stdInPipe = nil
	}
}

func (pr *ExternalProcess) Attach() error {
	app.At(te, "Attach")
	return pr.startRoutines()
}

func (pr *ExternalProcess) Detach() {
	app.At(te, "Detach")
	// TODO: detach using channels maybe
	pr.stopRoutines()
}

func (pr *ExternalProcess) GetId() msg.ExtProcessId {
	return pr.Id
}

func (pr *ExternalProcess) GetFullCommandLine() string {
	return pr.CommandLine
}

func (pr *ExternalProcess) GetProcessInChannel() chan []byte {
	return pr.ProcessIn
}

func (pr *ExternalProcess) GetProcessOutChannel() chan []byte {
	return pr.ProcessOut
}

func (pr *ExternalProcess) GetProcessExitChannel() chan struct{} {
	return pr.ProcessExit
}
