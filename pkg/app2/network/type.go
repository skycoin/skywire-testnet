package network

// Type represents the network type.
type Type string

const (
	// TypeDMSG is a network type for DMSG communication.
	TypeDMSG Type = "dmsg"
)

// IsValid checks whether the network contains valid value for the type.
func (n Type) IsValid() bool {
	_, ok := validNetworks[n]
	return ok
}

var (
	validNetworks = map[Type]struct{}{
		TypeDMSG: {},
	}
)
