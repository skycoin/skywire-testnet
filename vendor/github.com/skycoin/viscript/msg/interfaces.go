package msg

const ChannelCapacity = 4096 // FIXME?  might only need capacity of 2?
// .... onChar is always paired with an immediate onKey, making 2 entries at once

type ProcessInterface interface {
	GetId() ProcessId
	GetType() ProcessType
	GetLabel() string
	GetIncomingChannel() chan []byte //channel for incoming messages
	Tick()                           //process the messages and emit messages
}

type ExtProcessInterface interface {
	Tick()
	Start() error
	Attach() error
	Detach()
	TearDown()
	GetId() ExtProcessId
	GetFullCommandLine() string
	GetProcessInChannel() chan []byte
	GetProcessOutChannel() chan []byte
	GetProcessExitChannel() chan struct{}
}
