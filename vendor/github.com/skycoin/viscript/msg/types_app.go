package msg

const CATEGORY_App uint16 = 0x0300 //flag

const (
	TypeUserCommand        = 1 + CATEGORY_App
	TypeUserCommandAck     = 2 + CATEGORY_App  // Meshnet -> Viscript
	TypePing               = 3 + CATEGORY_App  // Viscript -> Meshnet
	TypePingAck            = 4 + CATEGORY_App  // Meshnet -> Viscript
	TypeCreateAck          = 5 + CATEGORY_App  // Meshnet -> Viscript
	TypeResourceUsage      = 6 + CATEGORY_App  // Viscript -> Meshnet
	TypeResourceUsageAck   = 7 + CATEGORY_App  // Meshnet -> Viscript
	TypeShutdown           = 8 + CATEGORY_App  // Viscript -> Meshnet
	TypeShutdownAck        = 9 + CATEGORY_App  // Meshnet -> Viscript
	TypeConnectDirectly    = 10 + CATEGORY_App // Viscript -> Meshnet
	TypeConnectDirectlyAck = 11 + CATEGORY_App // Meshnet -> Viscript
)

type MessageUserCommand struct {
	Sequence uint32
	AppId    uint32
	Payload  []byte
}

type MessageUserCommandAck struct {
	Sequence uint32
	AppId    uint32
	Payload  []byte
}

type MessageCreateAck struct{}

type MessageResourceUsage struct{}

type MessageResourceUsageAck struct {
	CPU    float64
	Memory uint64
}

type MessageShutdown struct{}

type MessageShutdownAck struct{}

type MessageConnectDirectly struct {
	Address string
}

type MessageConnectDirectlyAck struct{}

type MessagePing struct{}

type MessagePingAck struct{}
