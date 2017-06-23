package terminalmanager

import (
	"github.com/skycoin/viscript/hypervisor"
	"github.com/skycoin/viscript/hypervisor/dbus"
	"github.com/skycoin/viscript/viewport/terminal"
)

type TerminalManager struct {
	dbus          *dbus.DbusInstance
	terminalStack *terminal.TerminalStack
	processList   *hypervisor.ProcessList
}

func newTerminalManager() *TerminalManager {
	ntm := new(TerminalManager)
	ntm.dbus = &hypervisor.DbusGlobal
	ntm.terminalStack = &terminal.Terms
	ntm.processList = &hypervisor.ProcessListGlobal
	return ntm
}
