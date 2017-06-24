package cGfx

//"fmt"

//"github.com/skycoin/viscript/ui"

// func UpdateTextBody(body []string, colors []ColorSpot, content app.Rectangle) {
// cX := CurrX // current drawing position
// cY := CurrY
// cW := CharWid
// cH := CharHei
// b := content.Bottom

// setup for colored text
//ncId := 0         // next color
//var nc *ColorSpot // ^
// if /* colors exist */ len(colors) > 0 {
//nc = colors[ncId]
// }

// iterate over lines
// for _ /*y*/, line := range body {
// 	lineVisible := cY <= content.Top+cH && cY >= b

// 	if lineVisible {
// 		r := &app.PicRectangle{0, 0, 0, Pic_GradientBorder, &app.Rectangle{cY, cX + cW, cY - cH, cX}} // t, r, b, l

// 		// if line needs vertical adjustment
// 		if cY > content.Top {
// 			r.Top = content.Top
// 		}
// 		if cY-cH < b {
// 			r.Bottom = b
// 		}

// 		// iterate over runes
// 		//cGfx.SetColor(cGfx.Gray)
// 		for x, c := range line {
// 			//ncId, nc = t.changeColorIfCodeAt(x, y, ncId, nc)

// 			// drawing
// 			if /* char visible */ cX >= content.Left-cW && cX < content.Right {
// 				app.ClampLeftAndRightOf(r.Rectangle, content.Left, content.Right)
// 				//DrawCharAtRect(c, r.Rectangle)

// 				// if t.IsEditable {
// 				// 	if x == t.CursX && y == t.CursY {
// 				// 		gfx.SetColor(gfx.White)
// 				// 		gfx.Update9SlicedRect(Curs.GetAnimationModifiedRect(*r))
// 				// 		gfx.SetColor(gfx.PrevColor)
// 				// 	}
// 				// }
// 			}

// 			cX += cW
// 			r.Left = cX
// 			r.Right = cX + cW
// 		}

// 		// draw cursor at the end of line if needed
// 		//if cX < content.Right && y == t.CursY && t.CursX == len(line) {
// 		// if t.IsEditable {
// 		// 	//gfx.SetColor(gfx.White)
// 		// 	app.ClampLeftAndRightOf(r.Rectangle, content.Left, content.Right)
// 		// 	gfx.Update9SlicedRect(cGfx.Curs.GetAnimationModifiedRect(*r))
// 		// }
// 		//}

// 		//cX = GoToLeftEdge()
// 	} else { // line not visible
// 		for x := range line {
// 			//ncId, nc = t.changeColorIfCodeAt(x, y, ncId, nc)
// 		}
// 	}

// 	cY -= cH // go down a line height
//}
// }
