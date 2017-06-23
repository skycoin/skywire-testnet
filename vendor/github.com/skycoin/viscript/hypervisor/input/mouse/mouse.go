package mouse

import (
	"math"

	"github.com/skycoin/viscript/app"
)

var (
	GlPos               app.Vec2F //current mouse position in OpenGL space
	PrevGlPos           app.Vec2F //previous " " " " "
	PixelDelta          app.Vec2F //used when determining new scrollbar position (in old text editor)
	DeltaSinceLeftClick app.Vec2F
	LeftButtonIsDown    bool

	// private
	pixelSize_    app.Vec2F
	prevPixelPos  app.Vec2F
	canvasExtents app.Vec2F
	nearThresh    float64 //nearness threshold (how close pointer should be to the edge)
)

func Update(pixelPos app.Vec2F) {
	PrevGlPos = GlPos
	setGlPosFrom(pixelPos)
	DeltaSinceLeftClick.MoveBy(GlPos.GetDeltaFrom(PrevGlPos))
	PixelDelta = pixelPos.GetDeltaFrom(prevPixelPos)
	prevPixelPos.SetTo(pixelPos)
}

func NearRight(bounds *app.Rectangle) bool {
	return math.Abs(float64(GlPos.X-bounds.Right)) <= nearThresh
}

func NearBottom(bounds *app.Rectangle) bool {
	return math.Abs(float64(GlPos.Y-bounds.Bottom)) <= nearThresh
}

func IncreaseNearnessThreshold() {
	nearThresh = 10.0
}

func DecreaseNearnessThreshold() {
	nearThresh = 0.05
}

func PointerIsInside(r *app.Rectangle) bool {
	if GlPos.Y <= r.Top && GlPos.Y >= r.Bottom {
		if GlPos.X <= r.Right && GlPos.X >= r.Left {
			return true
		}
	}

	return false
}

func SetSizes(extents, pixelSize app.Vec2F) {
	canvasExtents = extents
	pixelSize_ = pixelSize
}

func GetScrollDeltaX() float32 {
	return PixelDelta.X * pixelSize_.X
}

func GetScrollDeltaY() float32 {
	return PixelDelta.Y * pixelSize_.Y
}

func setGlPosFrom(pos app.Vec2F) {
	GlPos.X = -canvasExtents.X + pos.X*pixelSize_.X
	GlPos.Y = canvasExtents.Y - pos.Y*pixelSize_.Y
}
