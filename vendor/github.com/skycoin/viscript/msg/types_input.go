package msg

const CATEGORY_Input uint16 = 0x0100 //flag

const (
	TypeMousePos    = 1 + CATEGORY_Input
	TypeMouseScroll = 2 + CATEGORY_Input
	TypeMouseButton = 3 + CATEGORY_Input
	TypeChar        = 4 + CATEGORY_Input
	TypeKey         = 5 + CATEGORY_Input
)

type MessageMousePos struct {
	X float64
	Y float64
}

type MessageMouseScroll struct {
	X              float64
	Y              float64
	HoldingControl bool
}

type MessageMouseButton struct {
	Button uint8
	Action uint8
	Mod    uint8
}

type MessageChar struct {
	Char uint32
}

type MessageKey struct {
	Key    uint32
	Scan   uint32
	Action uint8
	Mod    uint8
}
