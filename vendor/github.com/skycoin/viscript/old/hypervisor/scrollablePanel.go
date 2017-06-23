package hypervisor

import (
	"fmt"

	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/old/tree"
	"github.com/skycoin/viscript/old/ui"
	// "github.com/skycoin/viscript/cGfx"
	// "github.com/skycoin/viscript/tree"
	// "github.com/skycoin/viscript/ui"
	//"math"
)

type Terminal struct {
	FractionOfStrip float32 // fraction of the parent PanelStrip (in 1 dimension)
	CursX           int     // current cursor/insert position (in character grid cells/units)
	CursY           int
	MouseX          int // current mouse position in character grid space (units/cells)
	MouseY          int
	IsEditable      bool // editing is hardwired to TextBodies[0], but we probably never want
	// to edit text unless the whole panel is dedicated to just one TextBody (& no graphical trees)
	Whole      *app.Rectangle    // the whole panel, including chrome (title bar & scroll bars)
	Content    *app.PicRectangle // viewport into virtual space, subset of the Whole rect
	Virtual    *app.Rectangle    // virtual space, which can be less or more than Whole/Content, in both dimensions
	Selection  *ui.SelectionRange
	BarHori    *ui.ScrollBar // horizontal
	BarVert    *ui.ScrollBar // vertical
	TextBodies [][]string
	// TextColors []*cGfx.ColorSpot
	Trees []*tree.Tree
}

func (t *Terminal) Init() {
	fmt.Printf("<Terminal>.Init()\n")

	t.TextBodies = append(t.TextBodies, []string{})

	t.Selection = &ui.SelectionRange{}
	t.Selection.Init()

	// scrollbars
	t.BarHori = &ui.ScrollBar{IsHorizontal: true}
	t.BarVert = &ui.ScrollBar{}
	// t.BarHori.Rect = &app.PicRectangle{0, 0, 0, cGfx.Pic_GradientBorder, &app.Rectangle{}}
	// t.BarVert.Rect = &app.PicRectangle{0, 0, 0, cGfx.Pic_GradientBorder, &app.Rectangle{}}

	t.SetSize()
}

func (t *Terminal) SetSize() {
	fmt.Printf("<Terminal>.SetSize()\n")

	// t.Whole = &app.Rectangle{
	// cGfx.CanvasExtents.Y - cGfx.CharHei,
	// cGfx.CanvasExtents.X,
	// -cGfx.CanvasExtents.Y,
	// -cGfx.CanvasExtents.X}

	// if t.FractionOfStrip == runOutputTerminalFrac { // FIXME: this is hardwired for one use case for now
	// 	t.Whole.Top = t.Whole.Bottom + t.Whole.Height()*t.FractionOfStrip
	// } else {
	// 	t.Whole.Bottom = t.Whole.Bottom + t.Whole.Height()*runOutputTerminalFrac
	// }

	// t.Content = &app.PicRectangle{0, 0, 0, cGfx.Pic_GradientBorder, &app.Rectangle{}}
	// t.Content.Top = t.Whole.Top
	// t.Content.Right = t.Whole.Right - ui.ScrollBarThickness
	// t.Content.Bottom = t.Whole.Bottom + ui.ScrollBarThickness
	// t.Content.Left = t.Whole.Left

	// // set scrollbars' upper left corners
	// t.BarHori.Rect.Left = t.Whole.Left
	// t.BarHori.Rect.Top = t.Content.Bottom
	// t.BarVert.Rect.Left = t.Content.Right
	// t.BarVert.Rect.Top = t.Whole.Top
}

func (t *Terminal) RespondToMouseClick() {
	// Focused = t

	// // diffs/deltas from home position of panel (top left corner)
	// glDeltaFromHome := app.Vec2F{
	// 	mouse.GlX - t.Whole.Left,
	// 	mouse.GlY - t.Whole.Top}

	// t.MouseX = int((glDeltaFromHome.X + t.BarHori.ScrollDelta) / cGfx.CharWid)
	// t.MouseY = int(-(glDeltaFromHome.Y + t.BarVert.ScrollDelta) / cGfx.CharHei)

	// if t.MouseY < 0 {
	// 	t.MouseY = 0
	// }

	// if t.MouseY >= len(t.TextBodies[0]) {
	// 	t.MouseY = len(t.TextBodies[0]) - 1
	// }
}

// func (t *Terminal) changeColorIfCodeAt(x, y, ncId int, nc *cGfx.ColorSpot) (int, *cGfx.ColorSpot) {
// if /* colors exist */ len(t.TextColors) > 0 {
// 	if x == nc.Pos.X &&
// 		y == nc.Pos.Y {
// 		//gfx.SetColor(nc.Color)
// 		//fmt.Println("-------- nc-------, then 3rd():", nc, t.TextColors[2])
// 		ncId++

// 		if ncId < len(t.TextColors) {
// 			nc = t.TextColors[ncId]
// 		}
// 	}
// }

// return ncId, nc
// }

// func (t *Terminal) ScrollIfMouseOver(mousePixelDeltaX, mousePixelDeltaY float32) {
// 	if mouse.PointerIsInside(t.Whole) {
// position increments in gl space
// xInc := mousePixelDeltaX * cGfx.PixelSize.X
// yInc := mousePixelDeltaY * cGfx.PixelSize.Y
// 		t.BarHori.Scroll(xInc)
// 		t.BarVert.Scroll(yInc)
// 	}
// }

func (t *Terminal) RemoveCharacter(fromUnderCursor bool) {
	txt := t.TextBodies[0]

	if fromUnderCursor {
		if len(txt[t.CursY]) > t.CursX {
			txt[t.CursY] = txt[t.CursY][:t.CursX] + txt[t.CursY][t.CursX+1:len(txt[t.CursY])]
		}
	} else {
		if t.CursX > 0 {
			txt[t.CursY] = txt[t.CursY][:t.CursX-1] + txt[t.CursY][t.CursX:len(txt[t.CursY])]
			t.CursX--
		}
	}
}

func (t *Terminal) DrawTree() {
	// if len(t.Trees) > 0 {
	// 	// setup main rect
	// 	span := float32(1.3)
	// 	x := -span / 2
	// 	y := t.Whole.Top - 0.1
	// 	r := &app.Rectangle{y, x + span, y - span, x}

	// 	t.drawNodeAndDescendants(r, 0)
	// }
}

func (t *Terminal) drawNodeAndDescendants(r *app.Rectangle, nodeId int) {
	/*
		//fmt.Println("drawNode(r *app.Rectangle)")
		nameBar := &app.Rectangle{r.Top, r.Right, r.Top - 0.2*r.Height(), r.Left}
		cGfx.Update9SlicedRect(Pic_GradientBorder, r)
		SetColor(Blue)
		cGfx.Update9SlicedRect(Pic_GradientBorder, nameBar)
		DrawTextInRect(t.Trees[0].Nodes[nodeId].Text, nameBar)
		SetColor(White)

		cX := r.CenterX()
		rSp := r.Width() // rect span (height & width are the same)
		top := r.Bottom - rSp*0.5
		b := r.Bottom - rSp*1.5 // bottom

		node := t.Trees[0].Nodes[nodeId] // FIXME? .....
		// find t.Trees[0].Nodes[i].....
		// ......(if we ever use multiple trees per panel)
		// ......(also update DrawTree to use range)

		if node.ChildIdL != math.MaxInt32 {
			// (left child exists)
			x := cX - rSp*1.5
			t.drawArrowAndChild(r, &app.Rectangle{top, x + rSp, b, x}, node.ChildIdL)
		}

		if node.ChildIdR != math.MaxInt32 {
			// (right child exists)
			x := cX + rSp*0.5
			t.drawArrowAndChild(r, &app.Rectangle{top, x + rSp, b, x}, node.ChildIdR)
		}
	*/
}

func (t *Terminal) drawArrowAndChild(parent, child *app.Rectangle, childId int) {
	/*
		latExt := child.Width() * 0.15 // lateral extent of arrow's triangle top
		DrawTriangle(9, 1,
			app.Vec2F{parent.CenterX() - latExt, parent.Bottom},
			app.Vec2F{parent.CenterX() + latExt, parent.Bottom},
			app.Vec2F{child.CenterX(), child.Top})
		t.drawNodeAndDescendants(child, childId)
	*/
}

func (t *Terminal) SetupDemoProgram() {
	txt := []string{}

	txt = append(txt, "// ------- variable declarations ------- -------")
	//txt = append(txt, "var myVar int32")
	txt = append(txt, "var a int32 = 42 // end-of-line comment")
	txt = append(txt, "var b int32 = 58")
	txt = append(txt, "")
	txt = append(txt, "// ------- builtin function calls ------- ------- ------- ------- ------- ------- ------- end")
	txt = append(txt, "//    sub32(7, 9)")
	//txt = append(txt, "sub32(4,8)")
	//txt = append(txt, "mult32(7, 7)")
	//txt = append(txt, "mult32(3,5)")
	//txt = append(txt, "div32(8,2)")
	//txt = append(txt, "div32(15,  3)")
	//txt = append(txt, "add32(2,3)")
	//txt = append(txt, "add32(a, b)")
	txt = append(txt, "")
	txt = append(txt, "// ------- user function calls -------")
	txt = append(txt, "myFunc(a, b)")
	txt = append(txt, "")
	txt = append(txt, "// ------- function declarations -------")
	txt = append(txt, "func myFunc(a int32, b int32){")
	txt = append(txt, "")
	txt = append(txt, "        div32(6, 2)")
	txt = append(txt, "        innerFunc(a,b)")
	txt = append(txt, "}")
	txt = append(txt, "")
	txt = append(txt, "func innerFunc (a, b int32) {")
	txt = append(txt, "        var locA int32 = 71")
	txt = append(txt, "        var locB int32 = 29")
	txt = append(txt, "        sub32(locA, locB)")
	txt = append(txt, "}")

	/*
		for i := 0; i < 22; i++ {
			txt = append(txt, fmt.Sprintf("%d: put lots of text on screen", i))
		}
	*/

	t.TextBodies[0] = txt
}

func (t *Terminal) respondToVirtualSpaceResize() {
	// ************ FIXME these should be called when .Virtual space is implemented (& when it changes)
	// ************ FIXME also when physical space changes since bar is proportional to that
	//barHori.SetSize(t.Whole, t.TextBodies[0], CharWid, CharHei) // FIXME? (to consider multiple bodies & multiple trees)
	//barVert.SetSize(t.Whole, t.TextBodies[0], CharWid, CharHei)
}
