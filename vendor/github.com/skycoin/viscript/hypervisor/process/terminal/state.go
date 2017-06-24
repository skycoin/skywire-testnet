package process

import (
	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/msg"
)

var stPath = "hypervisor/process/terminal/state"

type State struct {
	DebugPrintInputEvents bool
	Cli                   *Cli
	VisualInfo            msg.MessageVisualInfo //dimensions, etc. (Terminal sends/updates)
	proc                  *Process
	storedTerminalIds     []msg.TerminalId
}

func (st *State) Init(proc *Process) {
	st.proc = proc
	st.DebugPrintInputEvents = true
	st.Cli = NewCli()
}

func (st *State) HandleMessages() {
	//called per Tick()
	c := st.proc.InChannel

	for len(c) > 0 {
		m := <-c
		//TODO/FIXME:   cache channel id wherever it may be needed
		m = m[4:] //.....for now, DISCARD the chan id prefix
		msgType := msg.GetType(m)
		msgCategory := msgType & 0xff00 // get back masked category

		switch msgCategory {

		case msg.CATEGORY_Input:
			st.UnpackMessage(msgType, m)
		case msg.CATEGORY_Terminal:
			st.UnpackMessage(msgType, m)
		default:
			app.At(stPath, "**************** UNHANDLED MESSAGE TYPE CATEGORY! ****************")

		}
	}
}
