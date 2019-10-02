package setup

import (
	"fmt"
)

type Failure struct {
	Code FailureCode `json:"code"`
	Msg  string      `json:"msg"`
}

func (f Failure) Error() string {
	return fmt.Sprintf("failure code %d (%s)", f.Code, f.Msg)
}

type FailureCode byte

// Failure codes
const (
	FailureUnknown FailureCode = iota
	FailureAddRules
	FailureCreateRoutes
	FailureRoutesCreated
	FailureReserveRtIDs
)

func (fc FailureCode) String() string {
	switch fc {
	case FailureUnknown:
		return "FailureUnknown"
	case FailureAddRules:
		return "FailureAddRules"
	case FailureCreateRoutes:
		return "FailureCreateRoutes"
	case FailureRoutesCreated:
		return "FailureRoutesCreated"
	case FailureReserveRtIDs:
		return "FailureReserveRtIDs"
	default:
		return fmt.Sprintf("unknown(%d)", fc)
	}
}
