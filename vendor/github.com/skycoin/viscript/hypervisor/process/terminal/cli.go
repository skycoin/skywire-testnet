package process

import (
	"strings"

	"github.com/skycoin/viscript/hypervisor"
	"github.com/skycoin/viscript/msg"
)

type Cli struct {
	Log      []string
	Commands []string
	CurrCmd  int //index
	CursPos  int //cursor/insert position, local to 1 commands space (2 lines)
	Prompt   string
	//FIXME to work with Terminal's dynamic self.GridSize.X
	//assumes 64 horizontal characters, then dedicates 2 lines for each command.
	BackscrollAmount int //number of VISUAL LINES...
	//(each could be merely a SECTION of a larger (than NumColumns) log entry)
	MaxCommandSize int //reserve ending space for cursor at the end of last line
}

func NewCli() *Cli {
	var cli Cli
	cli.Log = []string{}
	cli.Commands = []string{}
	cli.Prompt = ">"
	cli.Commands = append(cli.Commands, cli.Prompt+"OLDEST command that you typed (not really, just an example of functionality)")
	cli.Commands = append(cli.Commands, cli.Prompt+"older command that you typed (nah, not really)")
	cli.Commands = append(cli.Commands, cli.Prompt)
	cli.CursPos = 1
	cli.CurrCmd = 2
	cli.MaxCommandSize = 128 - 1

	return &cli
}

func (c *Cli) AdjustBackscrollOffset(delta int) {
	c.BackscrollAmount += delta

	if c.BackscrollAmount < 0 {
		c.BackscrollAmount = 0
	}

	//capping on the high end needs to be done dynamically
	//according to how many line sections/breaks there are
	println("BACKSCROLLING --- delta:", delta, " --- NUM:", c.BackscrollAmount)
}

func (c *Cli) InsertCharIfItFits(char uint32, state *State) {
	if len(c.Commands[c.CurrCmd]) < c.MaxCommandSize {
		c.InsertCharAtCursor(char)
		c.EchoWholeCommand(state.proc.OutChannelId)
	}
}

func (c *Cli) InsertCharAtCursor(char uint32) {
	c.Commands[c.CurrCmd] =
		c.Commands[c.CurrCmd][:c.CursPos] +
			string(char) +
			c.Commands[c.CurrCmd][c.CursPos:]
	c.moveCursorOneStepRight()
}

func (c *Cli) DeleteCharAtCursor() {
	c.Commands[c.CurrCmd] =
		c.Commands[c.CurrCmd][:c.CursPos] +
			c.Commands[c.CurrCmd][c.CursPos+1:]
}

func (c *Cli) EchoWholeCommand(outChanId uint32) {
	termId := uint32(0) //FIXME? correct terminal id really needed?
	//message := msg.Serialize(msg.TypePutChar, msg.MessagePutChar{0, m.Char})

	m := msg.Serialize(msg.TypeCommandLine,
		msg.MessageCommandLine{termId, c.Commands[c.CurrCmd], uint32(c.CursPos)})
	hypervisor.DbusGlobal.PublishTo(outChanId, m) //EVERY publish action prefixes another chan id
}

func (c *Cli) traverseCommands(delta int) {
	if delta > 1 || delta < -1 {
		println("FIXME if we ever want to stride/jump by more than 1")
		return
	}

	c.CurrCmd += delta

	if c.CurrCmd < 0 {
		c.CurrCmd = 0
	}

	if c.CurrCmd >= len(c.Commands) {
		c.CurrCmd = len(c.Commands) - 1
	}

	c.CursPos = len(c.Commands[c.CurrCmd])
}

func (c *Cli) moveCursorOneStepLeft() bool { //returns whether moved successfully
	c.CursPos--

	if c.CursPos < len(c.Prompt) {
		c.CursPos = len(c.Prompt)
		return false
	}

	return true
}

func (c *Cli) moveCursorOneStepRight() bool { //returns whether moved successfully
	c.CursPos++

	if c.CursPos > len(c.Commands[c.CurrCmd]) {
		c.CursPos = len(c.Commands[c.CurrCmd])
		return false
	} else if c.CursPos > c.MaxCommandSize { //allows cursor to be one position beyond last char
		c.CursPos = c.MaxCommandSize
		return false
	}

	return true
}

func (c *Cli) moveOrJumpCursorLeft(mod uint8) {
	if msg.ModifierKey(mod) == msg.ModControl {
		numSpaces := 0
		numVisible := 0 //NON-space

		for c.moveCursorOneStepLeft() == true {
			if c.Commands[c.CurrCmd][c.CursPos] == ' ' {
				numSpaces++

				if numVisible > 0 || numSpaces > 1 {
					c.moveCursorOneStepRight()
					break
				}
			} else {
				numVisible++
			}
		}
	} else {
		c.moveCursorOneStepLeft()
	}
}

func (c *Cli) moveOrJumpCursorRight(mod uint8) {
	if msg.ModifierKey(mod) == msg.ModControl {
		for c.moveCursorOneStepRight() == true {
			if c.CursPos < len(c.Commands[c.CurrCmd]) &&
				c.Commands[c.CurrCmd][c.CursPos] == ' ' {
				c.moveCursorOneStepRight()
				break
			}
		}
	} else {
		c.moveCursorOneStepRight()
	}
}

func (c *Cli) goUpCommandHistory(mod uint8) {
	if msg.ModifierKey(mod) == msg.ModControl {
		c.CurrCmd = 0
	} else {
		c.traverseCommands(-1)
	}
}

func (c *Cli) goDownCommandHistory(mod uint8) {
	if msg.ModifierKey(mod) == msg.ModControl {
		c.CurrCmd = len(c.Commands) - 1 //this could cause crash if we don't make sure at least 1 command always exists
	} else {
		c.traverseCommands(+1)
	}
}

func (c *Cli) CurrentCommandLine() string {
	return strings.ToLower(c.Commands[c.CurrCmd][len(c.Prompt):])
}

func (c *Cli) CurrentCommandAndArgs() (string, []string) {
	tokens := strings.Split(c.CurrentCommandLine(), " ")
	return tokens[0], tokens[1:]
}

func (c *Cli) OnEnter(st *State, serializedMsg []byte) {
	numPieces := 1 //each logical line entry may be broken (word wrapped) into more visible lines

	if c.CursPos >= int(st.VisualInfo.NumColumns) {
		numPieces++
	}

	for numPieces > 0 { //pass onKey to terminal (set to Enter),
		numPieces-- //for each visible piece of a line
		hypervisor.DbusGlobal.PublishTo(st.proc.OutChannelId, serializedMsg)
	}

	c.Log = append(c.Log, c.Commands[c.CurrCmd])
	c.Commands = append(c.Commands, c.Prompt)
	st.onUserCommand()
	c.CurrCmd = len(c.Commands) - 1
	c.CursPos = len(c.Commands[c.CurrCmd])
}
