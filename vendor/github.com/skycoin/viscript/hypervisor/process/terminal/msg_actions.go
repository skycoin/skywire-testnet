package process

import (
	"fmt"

	//"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/hypervisor"
	"github.com/skycoin/viscript/msg"
)

func (st *State) onMouseScroll(m msg.MessageMouseScroll, serializedMsg []byte) {
	if m.HoldingControl {
		hypervisor.DbusGlobal.PublishTo(st.proc.OutChannelId, serializedMsg)
	} else {
		st.Cli.AdjustBackscrollOffset(int(m.Y))
		st.makePageOfLog(st.VisualInfo)
	}
}

func (st *State) onChar(m msg.MessageChar) {
	//println("process/terminal/events.onChar()")
	st.Cli.InsertCharIfItFits(m.Char, st)
}

func (st *State) onKey(m msg.MessageKey, serializedMsg []byte) {
	switch msg.Action(m.Action) {

	case msg.Press: //one time, when key is first pressed
		//modifier key combos should never auto-repeat
		st.onOneTimeHotkeys(m)
		fallthrough
	case msg.Repeat: //constantly repeated for as long as key is pressed
		st.onRepeatableKeys(m, serializedMsg)
		st.Cli.EchoWholeCommand(st.proc.OutChannelId)

	case msg.Release:
		//most keys will probably never do anything upon release
	}
}

func (st *State) onTerminalIds(m msg.MessageTerminalIds) {
	st.storedTerminalIds = m.TermIds
	st.PrintLn(fmt.Sprintf("Terminals (%d) (* -> focused):", len(m.TermIds)))

	for index, termID := range m.TermIds {
		s := fmt.Sprintf("[%d] %d", index, termID)

		if termID == m.Focused {
			st.PrintLn("*" + s)
		} else {
			st.PrintLn(" " + s)
		}
	}

	st.Cli.EchoWholeCommand(st.proc.OutChannelId)
}

func (st *State) onOneTimeHotkeys(m msg.MessageKey) {
	switch m.Key {

	case msg.KeyC:
		if m.Mod == msg.GLFW_MOD_CONTROL {
			if !st.proc.HasExtProcessAttached() {
				return
			}

			//TODO: send the exit request i.e sigint or
			//just maybe kill the process?
		}

	case msg.KeyZ:
		if m.Mod == msg.GLFW_MOD_CONTROL {
			if !st.proc.HasExtProcessAttached() {
				return
			}

			st.PrintLn("Detaching External Process")
			st.proc.DetachExternalProcess()
		}

	}
}

func (st *State) onRepeatableKeys(m msg.MessageKey, serializedMsg []byte) {
	switch m.Key {

	case msg.KeyHome:
		st.Cli.CursPos = len(st.Cli.Prompt)
	case msg.KeyEnd:
		st.Cli.CursPos = len(st.Cli.Commands[st.Cli.CurrCmd])

	case msg.KeyUp:
		st.Cli.goUpCommandHistory(m.Mod)
	case msg.KeyDown:
		st.Cli.goDownCommandHistory(m.Mod)

	case msg.KeyLeft:
		st.Cli.moveOrJumpCursorLeft(m.Mod)
	case msg.KeyRight:
		st.Cli.moveOrJumpCursorRight(m.Mod)

	case msg.KeyBackspace:
		if st.Cli.moveCursorOneStepLeft() { //...succeeded
			st.Cli.DeleteCharAtCursor()
		}
	case msg.KeyDelete:
		if st.Cli.CursPos < len(st.Cli.Commands[st.Cli.CurrCmd]) {
			st.Cli.DeleteCharAtCursor()
		}

	case msg.KeyEnter:
		st.Cli.OnEnter(st, serializedMsg)

	}
}

func (st *State) onUserCommand() {
	cmd, args := st.Cli.CurrentCommandAndArgs()
	if len(cmd) < 1 {
		println("**** ERROR! ****   Command was empty!  Returning.")
		return
	}

	//if having multiple lines is such a bad thing, then you should have went all
	//the way and put it all on one line.  it is much easier to overlook,
	//and therefore fights against the whole purpose of printing it, however.
	s := "onUserCommand()      command: \"" + cmd + "\""

	for i, arg := range args {
		if i == 0 {
			s += "      args:"
		}

		s += " [" + arg + "]"
	}

	println(s)

	if st.proc.HasExtProcessAttached() {
		extProcInChannel := st.proc.attachedExtProcess.GetProcessInChannel()
		extProcInChannel <- []byte(st.Cli.CurrentCommandLine())
		return
	}

	//internal task handling
	switch cmd {
	case "?":
		fallthrough
	case "h":
		fallthrough
	case "help":
		if len(args) != 0 {
			st.commandAppHelp(args)
			break
		} else {
			st.commandHelp()
		}

	case "c":
		fallthrough
	case "cls":
		fallthrough
	case "clear":
		st.commandClearTerminal()

	case "ls":
		st.commandListExternalTasks(args)

	//attach external task to terminal task
	case "attach":
		st.commandAttach(args)

	case "r":
		fallthrough
	case "rpc":
		st.commandStart([]string{
			"-a", "go", "run", "rpc/cli/cli.go"})

	//start new external task, detached running in bg by default
	case "s":
		fallthrough
	case "start":
		st.commandStart(args)

	//ping app
	case "ping":
		st.commandAppPing(args)

	//shutdown running app
	case "shutdown":
		st.commandShutDown(args)

	//resource usage
	case "res_usage":
		st.commandResourceUsage(args)

	//add new terminal
	case "n":
		fallthrough
	case "new_term":
		st.SendCommand("new_term", []string{})

	//list all terminals marking focused with •
	case "list_terms":
		st.SendCommand("list_terms", []string{})

	//delete terminal with given index to the stored termIds
	case "delete_term":
		st.deleteTerminal(args)

	//display all apps with descriptions
	case "apps":
		st.commandDisplayApps()

	default:
		st.PrintError("\"" + cmd + "\" is an unknown command.")
	}
}
