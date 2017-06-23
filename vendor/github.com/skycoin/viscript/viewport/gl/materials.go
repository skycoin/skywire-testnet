package gl

import (
	"github.com/skycoin/viscript/app"
)

// pics (cell/tile positions in atlas)
var Pic_ArrowUp = app.Vec2I{8, 1}
var Pic_GradientBorder = app.Vec2I{11, 13}
var Pic_PixelCheckerboard = app.Vec2I{2, 11}
var Pic_SquareInTheMiddle = app.Vec2I{14, 15}
var Pic_DoubleLinesHorizontal = app.Vec2I{13, 12}
var Pic_DoubleLinesVertical = app.Vec2I{10, 11}
var Pic_DoubleLinesElbowBR = app.Vec2I{12, 11} // BR = bottom right

// colors
var Black = []float32{0, 0, 0, 1}
var Blue = []float32{0, 0, 1, 1}
var Cyan = []float32{0, 0.5, 1, 1}
var Fuschia = []float32{0.6, 0.2, 0.3, 1}
var Gray = []float32{0.25, 0.25, 0.25, 1}
var GrayDark = []float32{0.15, 0.15, 0.15, 1}
var GrayLight = []float32{0.4, 0.4, 0.4, 1}
var Green = []float32{0, 1, 0, 1}
var Magenta = []float32{1, 0, 1, 1}
var Maroon = []float32{0.5, 0.03, 0.207, 1}
var MaroonDark = []float32{0.24, 0.014, 0.1035, 1}
var Orange = []float32{0.8, 0.35, 0, 1}
var Purple = []float32{0.6, 0, 0.8, 1}
var Red = []float32{1, 0, 0, 1}
var Tan = []float32{0.55, 0.47, 0.37, 1}
var Violet = []float32{0.4, 0.2, 1, 1}
var White = []float32{1, 1, 1, 1}
var Yellow = []float32{1, 1, 0, 1}
