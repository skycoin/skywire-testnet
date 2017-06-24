package ui

import (
	//"fmt"
	"github.com/skycoin/viscript/app"
)

var ScrollBarThickness float32 = 0.14 // FUTURE FIXME: base this on screen size, so UHD+ users get thick enough bars

type ScrollBar struct {
	IsHorizontal   bool
	LenOfBar       float32
	LenOfVoid      float32           // length of the negative space representing the length of entire document
	LenOfOffscreen float32           // total hidden offscreen space (bookending the visible portal)
	ScrollDelta    float32           // distance/offset from home/start of document (negative in Y cuz Y increases upwards)
	VirtualWid     float32           // virtual space width.  allocating just enough to accommodate the longest line
	Rect           *app.PicRectangle // ... of the grabbable handle
}

func (bar *ScrollBar) Scroll(delta float32) {
	/*
		mouse position updates use pixels, so the smallest drag motions will be
		a jump of at least 1 pixel height.
		the ratio of that height / LenOfVoid (bar representing the page size),
		compared to the void/offscreen length of the text body,
		gives us the jump size in scrolling through the text body
	*/

	amount := delta / bar.LenOfVoid * bar.LenOfOffscreen

	if bar.IsHorizontal {
		bar.ScrollDelta += amount
		bar.ClampX()
	} else { // vertical
		bar.ScrollDelta -= amount
		bar.ClampY()
	}
}

func (bar *ScrollBar) ClampX() {
	bar.ScrollDelta = app.Clamp(bar.ScrollDelta, 0, bar.LenOfOffscreen)
}

func (bar *ScrollBar) ClampY() {
	bar.ScrollDelta = app.Clamp(bar.ScrollDelta, -bar.LenOfOffscreen, 0)
}

func (bar *ScrollBar) SetSize(panel *app.Rectangle, body []string, charWid, charHei float32) {
	if bar.IsHorizontal {
		// OPTIMIZEME in the future?  idealistically, the below should only be calculated
		// whenever user changes the size of a line, such as by:
		// 		typing/deleting/hitting-enter (could be splitting the biggest line in 2).
		// or changes client window size, or panel size
		// ....but it probably will never really matter.

		// find....
		numCharsInLongest := 0 // ...line of the document
		for _, line := range body {
			if numCharsInLongest < len(line) {
				numCharsInLongest = len(line)
			}
		}
		numCharsInLongest++ // adding extra space so we have room to show cursor at end of longest

		// the rest of this block is an altered copy of the else block
		panWid := panel.Width() - ScrollBarThickness // width of panel (MINUS scrollbar space)

		/* if content smaller than panel width */
		if float32(numCharsInLongest)*charWid <= panel.Width()-ScrollBarThickness {
			// NO BAR
			bar.LenOfBar = 0
			bar.LenOfVoid = panWid
			bar.LenOfOffscreen = 0
		} else { // need bar
			bar.VirtualWid = float32(numCharsInLongest) * charWid
			bar.LenOfOffscreen = bar.VirtualWid - panWid
			bar.LenOfBar = panWid / bar.VirtualWid * panWid
			bar.LenOfVoid = panWid - bar.LenOfBar
			bar.Rect.Left = panel.Left + bar.ScrollDelta/bar.LenOfOffscreen*bar.LenOfVoid
			bar.ClampX() // OPTIMIZEME: only do when app resized
		}
	} else { // vertical bar
		panHei := panel.Height() - ScrollBarThickness // height of panel (MINUS scrollbar space)

		/* if content smaller than panel height */
		if float32(len(body))*charHei <= panel.Height()-ScrollBarThickness {
			// NO BAR
			bar.LenOfBar = 0
			bar.LenOfVoid = panHei
			bar.LenOfOffscreen = 0
		} else { // need bar
			totalTextHei := float32(len(body)) * charHei
			//fmt.Printf("totalTextHei: %.2f\n", totalTextHei)
			bar.LenOfOffscreen = totalTextHei - panHei
			//fmt.Printf("LenOfOffscreen: %.2f\n", bar.LenOfOffscreen)
			bar.LenOfBar = panHei / totalTextHei * panHei
			//fmt.Printf("LenOfBar: %.2f\n", bar.LenOfBar)
			bar.LenOfVoid = panHei - bar.LenOfBar
			//fmt.Printf("LenOfVoid: %.2f\n", bar.LenOfVoid)
			bar.Rect.Top = panel.Top + bar.ScrollDelta/bar.LenOfOffscreen*bar.LenOfVoid
			bar.ClampY() // OPTIMIZEME: only do when app resized
		}
	}

	// setup bottom right corner of final rectangle for drawing
	if bar.IsHorizontal {
		bar.Rect.Right = bar.Rect.Left + bar.LenOfBar
		bar.Rect.Bottom = panel.Bottom
	} else {
		bar.Rect.Bottom = bar.Rect.Top - bar.LenOfBar
		bar.Rect.Right = panel.Right
	}
}
