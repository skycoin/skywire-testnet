package app

type Vec2I struct {
	X int
	Y int
}

type Vec2UI32 struct {
	X uint32
	Y uint32
}

type Vec2F struct {
	X float32
	Y float32
}

func (v *Vec2F) SetTo(value Vec2F) {
	v.X = value.X
	v.Y = value.Y
}

func (v *Vec2F) MoveBy(amount Vec2F) {
	v.X += amount.X
	v.Y += amount.Y
}

func (v *Vec2F) GetDeltaFrom(point Vec2F) Vec2F {
	return Vec2F{
		v.X - point.X,
		v.Y - point.Y}
}
