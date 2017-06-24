package gl

import (
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/skycoin/viscript/msg"
)

func InitMiscEvents(w *glfw.Window) {
	w.SetFramebufferSizeCallback(onFrameBufferSize)
}

func onFrameBufferSize(w *glfw.Window, width, height int) {
	msg.SerializeAndDispatch(
		InputEvents,
		msg.TypeFrameBufferSize,
		msg.MessageFrameBufferSize{uint32(width), uint32(height)})
}
