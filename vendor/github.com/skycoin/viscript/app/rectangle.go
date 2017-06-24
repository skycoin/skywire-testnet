package app

type Rectangle struct {
	Top    float32
	Right  float32
	Bottom float32
	Left   float32
}

func (r *Rectangle) Width() float32 {
	return r.Right - r.Left
}

func (r *Rectangle) Height() float32 {
	return r.Top - r.Bottom
}

func (r *Rectangle) CenterX() float32 {
	return r.Left + r.Width()/2
}

func (r *Rectangle) CenterY() float32 {
	return r.Bottom + r.Height()/2
}

func (r *Rectangle) Contains(x, y float32) bool {
	if x > r.Left && x < r.Right && y > r.Bottom && y < r.Top {
		return true
	}
	return false
}

func (r *Rectangle) MoveBy(amount Vec2F) {
	r.Top += amount.Y
	r.Bottom += amount.Y
	r.Left += amount.X
	r.Right += amount.X
}

// ----------------------------------
// rectangles with extra graphic data
// ----------------------------------

// rectangle types
const (
	RectType_9Slice = iota
	RectType_Simple // only requires one quad which is uniformly shrunk or stretched
)

// state
const (
	RectState_Active   = iota
	RectState_Inactive // .../invisible
	RectState_Dead
)

type PicRectangle struct {
	Id    int32
	Type  uint8
	State uint8
	//Color    float32
	AtlasPos Vec2I // x, y position in the atlas
	*Rectangle
}

type NineSliceRectangle struct {
	XSpots [3]float32 // positions of vertical lines
	YSpots [3]float32 // positions of horizontal lines
	USpots [3]float32 // texture coords of vertical lines
	VSpots [3]float32 // texture coords of horizontal lines
	*PicRectangle
}
