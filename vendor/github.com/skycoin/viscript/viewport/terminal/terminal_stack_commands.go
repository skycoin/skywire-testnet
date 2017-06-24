package terminal

import (
	"strconv"

	"github.com/skycoin/viscript/msg"
)

func (ts *TerminalStack) OnUserCommand(tID msg.TerminalId, cmd msg.MessageCommand) {
	switch cmd.Command {

	case "new_term":
		ts.Add()
	case "list_terms":
		ts.ListTerminalsWithIds(tID)
	case "delete_term":
		if len(cmd.Args) != 1 {
			return
		}

		ts.DeleteTerminalIfExists(cmd.Args[0])
	default:

	}
}

func (ts *TerminalStack) ListTerminalsWithIds(termId msg.TerminalId) {
	var m msg.MessageTerminalIds
	m.Focused = termId

	for _, term := range ts.Terms {
		m.TermIds = append(m.TermIds, term.TerminalId)
	}

	ts.Focused.RelayToTask(msg.Serialize(msg.TypeTerminalIds, m))
}

func (ts *TerminalStack) DeleteTerminalIfExists(termIdString string) {
	termIdInt, err := strconv.Atoi(termIdString)
	if err != nil {
		return
	}

	ts.RemoveTerminal(msg.TerminalId(termIdInt))
}
