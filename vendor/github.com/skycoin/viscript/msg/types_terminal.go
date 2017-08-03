package msg

const CATEGORY_Terminal uint16 = 0x0200 //flag

const (
	TypeClear            = 1 + CATEGORY_Terminal
	TypeTokenizedCommand = 2 + CATEGORY_Terminal
	TypeCommandPrompt    = 3 + CATEGORY_Terminal
	TypePutChar          = 4 + CATEGORY_Terminal
	TypeSetCharAt        = 5 + CATEGORY_Terminal
	TypeSetCursor        = 6 + CATEGORY_Terminal
	TypeTerminalIds      = 7 + CATEGORY_Terminal
	TypeVisualInfo       = 8 + CATEGORY_Terminal
	TypeFrameBufferSize  = 9 + CATEGORY_Terminal //start of low level events
)

type MessageClear struct { //this type simply signals that we need a .clear() call in terminal
}

type MessageTokenizedCommand struct {
	Command string
	Args    []string
}

type MessageCommandPrompt struct { //updates/replaces current command prompt on any change
	TermId       uint32
	CommandLine  string
	CursorOffset uint32 //from first character of command prompt
}

type MessagePutChar struct {
	TermId uint32
	Char   uint32
}

type MessageSetCharAt struct {
	TermId uint32
	X      uint32
	Y      uint32
	Char   uint32
}

type MessageSetCursor struct {
	TermId uint32
	X      uint32
	Y      uint32
}

type MessageTerminalIds struct {
	Focused TerminalId
	TermIds []TerminalId
}

type MessageVisualInfo struct {
	NumColumns uint32
	NumRows    uint32
	CurrColumn uint32
	CurrRow    uint32
	PromptRows uint32
}

//low level events
type MessageFrameBufferSize struct {
	X uint32
	Y uint32
}
