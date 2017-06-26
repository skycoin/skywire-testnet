package msg

import (
	"math/rand"
)

type ProcessId uint64 //HyperVisor: processId
type TerminalId uint64
type ExtProcessId uint64

var ProcessIdGlobal ProcessId = 1 //sequential
var ExtProcessIdGlobal ExtProcessId = 1

func NextProcessId() ProcessId {
	ProcessIdGlobal += 1
	return ProcessIdGlobal
}

func NextExtProcessId() ExtProcessId {
	ExtProcessIdGlobal += 1
	return ExtProcessIdGlobal
}

func RandTerminalId() TerminalId {
	return (TerminalId)(rand.Int63())
}
