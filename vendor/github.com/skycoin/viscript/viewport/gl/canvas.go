// canvas == the whole "client" area of the graphical OpenGL window (of root app)
package gl

import (
	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/hypervisor/input/mouse"
)

var (
	// distance from the center to top & bottom edges of the canvas
	DistanceFromOrigin float32 = 1

	// dimensions (in pixel units)
	InitAppWidth  int = 1024 // initial/startup size (when resizing, compare against this)
	InitAppHeight int = 768
	CurrAppWidth      = int32(InitAppWidth) // current
	CurrAppHeight     = int32(InitAppHeight)

	// GL space floats
	longerDimension = float32(InitAppWidth) / float32(InitAppHeight)
	InitFrustum     = &app.Rectangle{1, longerDimension, -1, -longerDimension}
	PrevFrustum     = &app.Rectangle{InitFrustum.Top, InitFrustum.Right, InitFrustum.Bottom, InitFrustum.Left}
	CurrFrustum     = &app.Rectangle{InitFrustum.Top, InitFrustum.Right, InitFrustum.Bottom, InitFrustum.Left}

	// to give us proportions similar to a text mode command prompt screen
	NumChars = &app.Vec2I{80, 25}
)

var (
	CanvasExtents app.Vec2F
	PixelSize     app.Vec2F
	CharWid       float32
	CharHei       float32

	// current position renderer draws to
	CurrX float32
	CurrY float32
)

func Init() {
	println("<gl/canvas>.Init()")
	// one-time setup
	PrevColor = GrayDark
	CurrColor = GrayDark

	// things that are resized later
	CanvasExtents.X = DistanceFromOrigin * longerDimension
	CanvasExtents.Y = DistanceFromOrigin
	CharWid = float32(CanvasExtents.X*2) / float32(NumChars.X)
	CharHei = float32(CanvasExtents.Y*2) / float32(NumChars.Y)
	PixelSize.X = CanvasExtents.X * 2 / float32(CurrAppWidth)
	PixelSize.Y = CanvasExtents.Y * 2 / float32(CurrAppHeight)

	// MORE one-time setup
	MainMenu.SetSize(GetTaskbarRect())
	mouse.SetSizes(CanvasExtents, PixelSize)
}

func GetTaskbarRect() *app.Rectangle {
	return &app.Rectangle{
		CanvasExtents.Y,
		CanvasExtents.X,
		CanvasExtents.Y - CharHei,
		-CanvasExtents.X}
}

func SetSize(x, y int32) {
	println("\n<gl/canvas>.SetSize()", x, y)

	*PrevFrustum = *CurrFrustum
	CurrAppWidth = x
	CurrAppHeight = y

	CurrFrustum.Right = float32(CurrAppWidth) / float32(InitAppWidth) * InitFrustum.Right
	CurrFrustum.Left = -CurrFrustum.Right
	CurrFrustum.Top = float32(CurrAppHeight) / float32(InitAppHeight) * InitFrustum.Top
	CurrFrustum.Bottom = -CurrFrustum.Top

	CanvasExtents.X = DistanceFromOrigin * CurrFrustum.Right
	CanvasExtents.Y = DistanceFromOrigin * CurrFrustum.Top

	// things that weren't initialized in this func
	MainMenu.SetSize(GetTaskbarRect())
	mouse.SetSizes(CanvasExtents, PixelSize)
}
