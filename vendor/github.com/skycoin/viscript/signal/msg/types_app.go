package msg

const CATEGORY_App uint16 = 0x0300 //flag

const (
	TypeUserCommand        = 1 + CATEGORY_App
	TypeUserCommandAck     = 2 + CATEGORY_App  // Meshnet -> Viscript
	TypePing               = 3 + CATEGORY_App  // Viscript -> Meshnet
	TypePingAck            = 4 + CATEGORY_App  // Meshnet -> Viscript
	TypeResourceUsage      = 5 + CATEGORY_App  // Viscript -> Meshnet
	TypeResourceUsageAck   = 6 + CATEGORY_App  // Meshnet -> Viscript
	TypeShutdown           = 7 + CATEGORY_App  // Viscript -> Meshnet
	TypeShutdownAck        = 8 + CATEGORY_App  // Meshnet -> Viscript
	TypeStartup            = 9 + CATEGORY_App  // Viscript -> Meshnet
	TypeStartupAck         = 10 + CATEGORY_App  // Meshnet -> Viscript
	TypeFirstConnect       = 11 + CATEGORY_App  // Viscript -> Meshnet

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

type MessageResourceUsage struct{}

type MessageResourceUsageAck struct {
	CPU    float64
	Memory uint64
}

type MessageShutdown struct{
	Stage uint32
}

type MessageShutdownAck struct{
	Stage uint32
}

type MessagePing struct{}

type MessagePingAck struct{}

type MessageStartup struct{
	Address string
	Stage uint32
}

type MessageStartupAck struct{
	Address string
	Stage uint32
}

type MessageFirstConnect struct{
	Address string
	Port	string
}
