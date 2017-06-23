package viewport

import (
	"runtime"

	igl "github.com/skycoin/viscript/viewport/gl"         //internal gl
	stack "github.com/skycoin/viscript/viewport/terminal" //TerminalStack
)

//glfw
//glfw.PollEvents()
//only remaining

var CloseWindow bool = false

func Init() {
	println("<viewport>.Init()")

	//GLFW event handling must run on the main OS thread
	//See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()

	initScreen()
	initEvents()
	stack.Terms.Init()
}

func initScreen() {
	igl.Init() //in canvas.go
	igl.InitGlfw()
	igl.LoadTextures()
	igl.InitRenderer()
}

func initEvents() {
	igl.InitInputEvents(igl.GlfwWindow)
	igl.InitMiscEvents(igl.GlfwWindow)
}

func TeardownScreen() {
	println("<viewport>.TeardownScreen()")
	igl.ScreenTeardown()
}

func PollUiInputEvents() {
	igl.PollEvents() //move to gl
}

//could be in messages
func DispatchEvents() []byte {
	message := []byte{}

	for len(igl.InputEvents) > 0 {
		v := <-igl.InputEvents
		message = UnpackMessage(v)
	}

	return message
}

func Tick() {
	igl.Curs.Tick()
	stack.Terms.Tick()
}

func UpdateDrawBuffer() {
	igl.DrawBegin()
	stack.Terms.Draw()
	igl.DrawEnd()
}

func SwapDrawBuffer() {
	igl.SwapDrawBuffer()
}
