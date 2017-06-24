package cGfx

//"fmt"

// "github.com/skycoin/viscript/ui"

// func DrawMenu() {
// 	for _, bu := range ui.MainMenu.Buttons {
// 		if bu.Activated {
// 			//SetColor(cGfx.Green)
// 		} else {
// 			//SetColor(cGfx.White)
// 		}

// 		Update9SlicedRect(bu.Rect)
// 		//gl.DrawTextInRect(bu.Name, bu.Rect.Rectangle)
// 	}
// }

// func GoToTopEdge(r *app.PicRectangle, scrollDelta float32) {
// 	CurrY = r.Top - /*t.BarVert.S*/ scrollDelta
// }

// func GoToLeftEdge(r *app.PicRectangle, scrollDelta float32) float32 {
// 	CurrX = r.Left - /*t.BarHori.S*/ scrollDelta
// 	return CurrX
// }

// func DrawTerminal(content *app.PicRectangle, h, v *ui.ScrollBar) {
// 	GoToTopEdge(content, v.ScrollDelta)
// 	GoToLeftEdge(content, h.ScrollDelta)
// 	DrawBackground(content)
// 	//UpdateTextBody()
// 	//gfx.SetColor(cGfx..GrayDark)

// 	// ATM the only different between the 2 funcs below is the top left corner (involving 3 vertices)
// 	DrawScrollbarBackdrop(Pic_DoubleLinesVertical, content.Right, content.Top /*+ui.ScrollBarThickness FIXME when we add title bar*/)
// 	DrawScrollbarBackdrop(Pic_DoubleLinesHorizontal, content.Left, content.Bottom)
// 	DrawScrollbarBackdrop(Pic_DoubleLinesElbowBR, content.Right, content.Bottom) // corner elbow piece
// 	//gfx.SetColor(cGfx..Gray)
// 	Update9SlicedRect(h.Rect)
// 	Update9SlicedRect(v.Rect)
// 	//gfx.SetColor(cGfx..White)
// 	//t.DrawTree()
// }

// func DrawScrollbarBackdrop(atlasCell app.Vec2I, l, top float32) { // l = left
// 	/*
// 		span := app.UvSpan
// 		u := float32(atlasCell.X) * span
// 		v := float32(atlasCell.Y) * span

// 		gl.Normal3f(0, 0, 1)

// 		// bottom left   0, 1
// 		gl.TexCoord2f(u, v+span)
// 		gl.Vertex3f(l, t.Whole.Bottom, 0)

// 		// bottom right   1, 1
// 		gl.TexCoord2f(u+span, v+span)
// 		gl.Vertex3f(t.Whole.Right, t.Whole.Bottom, 0)

// 		// top right   1, 0
// 		gl.TexCoord2f(u+span, v)
// 		gl.Vertex3f(t.Whole.Right, top, 0)

// 		// top left   0, 0
// 		gl.TexCoord2f(u, v)
// 		gl.Vertex3f(l, top, 0)
// 	*/
// }

// func DrawBackground(r *app.PicRectangle) {
// 	//gfx.SetColor(gfx.GrayDark)
// 	Update9SlicedRect(r)
// }

// func Update9SlicedRect(r *app.PicRectangle) {
// 	// 9 quads (like a tic-tac-toe grid, or a "#", which has 9 cells)
// 	// which keep a predictable frame/margin/edge undistorted,
// 	// while stretching the middle to fit the desired space

// 	w := r.Width()
// 	h := r.Height()

// 	//var uvEdgeFraction float32 = 0.125 // 1/8
// 	var uvEdgeFraction float32 = 0.125 / 2 // 1/16
// 	// we're gonna draw from top to bottom (positivemost to negativemost)

// 	sp /* span */ := app.UvSpan
// 	u := float32(r.AtlasPos.X) * sp
// 	v := float32(r.AtlasPos.Y) * sp

// 	// setup the 4 lines needed (for 3 spanning sections)
// 	uSpots := []float32{}
// 	uSpots = append(uSpots, (u))
// 	uSpots = append(uSpots, (u)+sp*uvEdgeFraction)
// 	uSpots = append(uSpots, (u+sp)-sp*uvEdgeFraction)
// 	uSpots = append(uSpots, (u + sp))

// 	vSpots := []float32{}
// 	vSpots = append(vSpots, (v))
// 	vSpots = append(vSpots, (v)+sp*uvEdgeFraction)
// 	vSpots = append(vSpots, (v+sp)-sp*uvEdgeFraction)
// 	vSpots = append(vSpots, (v + sp))

// 	edgeSpan := PixelSize.X * 4
// 	if edgeSpan > w/2 {
// 		edgeSpan = w / 2
// 	}

// 	xSpots := []float32{}
// 	xSpots = append(xSpots, r.Left)
// 	xSpots = append(xSpots, r.Left+edgeSpan)
// 	xSpots = append(xSpots, r.Right-edgeSpan)
// 	xSpots = append(xSpots, r.Right)

// 	edgeSpan = PixelSize.Y * 4
// 	if edgeSpan > h/2 {
// 		edgeSpan = h / 2
// 	}

// 	ySpots := []float32{}
// 	ySpots = append(ySpots, r.Top)
// 	ySpots = append(ySpots, r.Top-edgeSpan)
// 	ySpots = append(ySpots, r.Bottom+edgeSpan)
// 	ySpots = append(ySpots, r.Bottom)

// 	if ySpots[1] > ySpots[0] {
// 		ySpots[1] = ySpots[0]
// 	}

// 	SetRect(r)
// }
