package script

type Token struct {
	LexType int
	Value   string
}

type VarBool struct {
	Name  string
	Value bool
}

type VarInt32 struct {
	Name  string
	Value int32
}

type VarString struct {
	Name  string
	Value string
}

type CodeBlock struct {
	Name        string
	VarBools    []*VarBool
	VarInt32s   []*VarInt32
	VarStrings  []*VarString
	SubBlocks   []*CodeBlock
	Expressions []string
	Parameters  []string // unused atm
}
