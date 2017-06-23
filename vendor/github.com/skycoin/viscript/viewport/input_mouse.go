package viewport

import (
	"fmt"

	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/config"
	"github.com/skycoin/viscript/hypervisor/input/mouse"
	"github.com/skycoin/viscript/msg"
	"github.com/skycoin/viscript/viewport/gl"
	t "github.com/skycoin/viscript/viewport/terminal"
)

// triggered both by moving **AND*** by pressing buttons
func onMouseCursorPos(m msg.MessageMousePos) {
	if config.DebugPrintInputEvents() {
		fmt.Print("msg.TypeMousePos")
		showFloat64("X", m.X)
		showFloat64("Y", m.Y)
		println()
	}

	mouse.Update(app.Vec2F{float32(m.X), float32(m.Y)})

	foc := t.Terms.Focused

	if foc == nil {
		return
	}

	// set pointer appropriately
	if mouse.NearRight(foc.Bounds) && !foc.ResizingBottom && !mouse.LeftButtonIsDown {
		gl.SetHResizePointer()
	} else if mouse.NearBottom(foc.Bounds) && !foc.ResizingRight && !mouse.LeftButtonIsDown {
		gl.SetVResizePointer()
	} else if mouse.PointerIsInside(foc.Bounds) {
		gl.SetIBeamPointer()
	} else {
		gl.SetArrowPointer()
	}

	if mouse.LeftButtonIsDown {
		// Determination should be here if the mouse is over scrollbar or over the
		// area where terminal can be moved. Moving windows happens in GL space
		// coordinates because I thought pixel delta was used for scrollbar scrolling

		// REFACTORME: cause I made it messy i guess
		// FIXME: Also the context in this case text is left there and
		// allowed to write outside the bounds
		// should resize or it should be using characters as kind of measures

		if mouse.NearRight(foc.Bounds) && !foc.ResizingBottom {
			gl.SetHResizePointer()
			mouse.IncreaseNearnessThreshold()

			t.Terms.Focused.ResizeHorizontally(mouse.GlPos.X)
		} else if mouse.NearBottom(foc.Bounds) && !foc.ResizingRight {
			gl.SetVResizePointer()
			mouse.IncreaseNearnessThreshold()

			t.Terms.Focused.ResizeVertically(mouse.GlPos.Y)
		}

		if mouse.PointerIsInside(foc.Bounds) && !foc.IsResizing() {
			//high resolution delta for smooth resizing
			delt := mouse.GlPos.GetDeltaFrom(mouse.PrevGlPos)

			t.Terms.MoveFocusedTerminal(delt, &mouse.DeltaSinceLeftClick)
			gl.SetHandPointer()

			// if config.DebugPrintInputEvents() {
			// println("\nTerminal Id:", foc.TerminalId,
			// 	"\nTop", foc.Bounds.Top,
			// 	"\nLeft", foc.Bounds.Left,
			// 	"\nRight", foc.Bounds.Right,
			// 	"\nBottom", foc.Bounds.Bottom,
			// 	"\n\n GL MouseX:", mouse.GlPos.X,
			// 	"\n GL MouseY:", mouse.GlPos.Y,
			// 	"\n\n Previous GL MouseX:", mouse.PrevGlPos.X,
			// 	"\n Previous GL MouseY:", mouse.PrevGlPos.Y,
			// 	"\n\n delt.X:", delt.X,
			// 	"\n delt.Y:", delt.Y,
			// 	"\n\n Rect Center X:", foc.Bounds.CenterX(),
			// 	"\n Rect Center Y:", foc.Bounds.CenterY())
			// }
		}
	} else {
		foc.SetResizingOff()
		mouse.DecreaseNearnessThreshold()
	}
}

func onMouseScroll(m msg.MessageMouseScroll) {
	if config.DebugPrintInputEvents() {
		print("msg.TypeMouseScroll")
		showFloat64("X Offset", m.X)
		showFloat64("Y Offset", m.Y)
		showBool("HoldingControl", m.HoldingControl)
		println()
	}
}

// apparently every time this is fired, a mouse position event is ALSO fired
func onMouseButton(m msg.MessageMouseButton) {
	if config.DebugPrintInputEvents() {
		fmt.Print("msg.TypeMouseButton")
		showUInt8("Button", m.Button)
		showUInt8("Action", m.Action)
		showUInt8("Mod", m.Mod)
		println()
	}

	convertClickToTextCursorPosition(m.Button, m.Action)

	if msg.Action(m.Action) == msg.Press {
		switch msg.MouseButton(m.Button) {
		case msg.MouseButtonLeft:
			mouse.LeftButtonIsDown = true
			mouse.DeltaSinceLeftClick = app.Vec2F{0, 0}

			// // detect clicks in rects
			// if mouse.PointerIsInside(ui.MainMenu.Rect) {
			// 	respondToAnyMenuButtonClicks()
			// } else { // respond to any panel clicks outside of menu
			focusOnTopmostRectThatContainsPointer()
			// }
		}
	} else if msg.Action(m.Action) == msg.Release {
		switch msg.MouseButton(m.Button) {
		case msg.MouseButtonLeft:
			mouse.LeftButtonIsDown = false
		}
	}
}

func focusOnTopmostRectThatContainsPointer() {
	var topmostZ float32
	var topmostId msg.TerminalId

	for id, t := range t.Terms.Terms {
		if mouse.PointerIsInside(t.Bounds) {
			if topmostZ < t.Depth {
				topmostZ = t.Depth
				topmostId = id
			}
		}
	}

	if topmostZ > 0 {
		t.Terms.SetFocused(topmostId)
	}
}

func convertClickToTextCursorPosition(button, action uint8) {
	// if glfw.MouseButton(button) == glfw.MouseButtonLeft &&
	// 	glfw.Action(action) == glfw.Press {

	// 	foc := Focused

	// 	if foc.IsEditable && foc.Content.Contains(mouse.GlX, mouse.GlY) {
	// 		if foc.MouseY < len(foc.TextBodies[0]) {
	// 			foc.CursY = foc.MouseY

	// 			if foc.MouseX <= len(foc.TextBodies[0][foc.CursY]) {
	// 				foc.CursX = foc.MouseX
	// 			} else {
	// 				foc.CursX = len(foc.TextBodies[0][foc.CursY])
	// 			}
	// 		} else {
	// 			foc.CursY = len(foc.TextBodies[0]) - 1
	// 		}
	// 	}
	// }
}

func respondToAnyMenuButtonClicks() {
	// for _, bu := range ui.MainMenu.Buttons {
	// 	if mouse.PointerIsInside(bu.Rect.Rectangle) {
	// 		bu.Activated = !bu.Activated

	// 		switch bu.Name {
	// 		case "Run":
	// 			if bu.Activated {
	// 				//script.Process(true)
	// 			}
	// 			break
	// 		case "Testing Tree":
	// 			if bu.Activated {
	// 				//script.Process(true)
	// 				//script.MakeTree()
	// 			} else { // deactivated
	// 				// remove all terminals with trees
	// 				b := t.Terms[:0]
	// 				for _, t := range t.Terms {
	// 					if len(t.Trees) < 1 {
	// 						b = append(b, t)
	// 					}
	// 				}
	// 				t.Terms = b
	// 				//fmt.Printf("len of b (from Terms) after removing ones with trees: %d\n", len(b))
	// 				//fmt.Printf("len of Terms: %d\n", len(Terms))
	// 			}
	// 			break
	// 		}

	// 		app.Con.Add(fmt.Sprintf("%s toggled\n", bu.Name))
	// 	}
	// }
}

// the rest of these funcs are almost identical, just top 2 vars customized (and string format)
func showBool(s string, x bool) {
	fmt.Printf("   [%s: %t]", s, x)
}

func showUInt8(s string, x uint8) {
	fmt.Printf("   [%s: %d]", s, x)
}

func showSInt32(s string, x int32) {
	fmt.Printf("   [%s: %d]", s, x)
}

func showUInt32(s string, x uint32) {
	fmt.Printf("   [%s: %d]", s, x)
}

func showFloat64(s string, f float64) {
	fmt.Printf("   [%s: %.1f]", s, f)
}
