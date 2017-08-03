package msg

import (
	"math/rand"
)

var TaskIdGlobal TaskId = 1 //sequential
var ExtTaskIdGlobal ExtAppId = 1

type TaskId uint64
type ExtAppId uint64
type TerminalId uint64

//
//
//
func NextTaskId() TaskId {
	TaskIdGlobal += 1
	return TaskIdGlobal
}

func NextExtTaskId() ExtAppId {
	ExtTaskIdGlobal += 1
	return ExtTaskIdGlobal
}

func RandTerminalId() TerminalId {
	return (TerminalId)(rand.Int63())
}
