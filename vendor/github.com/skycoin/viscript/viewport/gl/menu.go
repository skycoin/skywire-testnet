package gl

import (
	"github.com/skycoin/viscript/app"
)

var MainMenu = Menu{}

type Menu struct {
	//IsVertical bool // controls which dimension gets divided up for button sizes
	Rect *app.Rectangle
	//Buttons    []*Button
}

func (m *Menu) SetSize(r *app.Rectangle) {
	m.Rect = r

	// depending on vertical or horizontal layout, only 1 dimension (of the below 4 variables) is used
	//x := m.Rect.Left
	//y := m.Rect.Top
	//wid := m.Rect.Width() / float32(len(m.Buttons))  // width of buttons
	//hei := m.Rect.Height() / float32(len(m.Buttons)) // height of buttons

}
