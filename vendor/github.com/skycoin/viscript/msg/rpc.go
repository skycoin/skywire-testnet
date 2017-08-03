package msg

type TaskInfo struct {
	Id    TaskId
	Type  TaskType
	Label string
}

//this is used to serialize and deserialize only these fields (to text, for user feedback)
type TermAndAttachedTaskId struct {
	TerminalId     TerminalId
	AttachedTaskId TaskId
}
