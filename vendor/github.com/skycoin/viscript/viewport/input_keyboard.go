package viewport

import (
	"fmt"

	"github.com/skycoin/viscript/config"
	"github.com/skycoin/viscript/hypervisor/input/keyboard"
	"github.com/skycoin/viscript/msg"
	"github.com/skycoin/viscript/viewport/gl"
	t "github.com/skycoin/viscript/viewport/terminal"
)

func onChar(m msg.MessageChar) {
	if config.DebugPrintInputEvents() {
		println("msg.TypeChar", " ["+string(m.Char)+"]") //if they want to see what events are triggered,
		//we shouldn't hide them without good reason (like the super spammy mouse moves)
	}
}

//WEIRD BEHAVIOUR OF KEY EVENTS.... for a PRESS, you can detect a
//shift/alt/ctrl/super key through the "mod" variable,
//(see the top of "action == glfw.Press" section for an example)
//regardless of left/right key used.
//BUT for RELEASE, the "mod" variable will NOT tell you what key it is!
//so you will have to handle both left & right modifier keys via the "action" variable!
func onKey(m msg.MessageKey) {
	if config.DebugPrintInputEvents() {
		if m.Action == 1 {
			println()
		}

		fmt.Printf("msg.TypeKey")
		showUInt32("Key", m.Key)
		showUInt32("Scan", m.Scan)
		showUInt8("Action", m.Action)
		showUInt8("Mod", m.Mod)
		println() //need for separation between event's feedback lines (they are composed without newlines)
	}

	if msg.Action(m.Action) == msg.Release {
		switch m.Key {

		case msg.KeyEscape:
			println("\n\nCLOSING OPENGL WINDOW")
			CloseWindow = true

		case msg.KeyLeftShift:
			fallthrough
		case msg.KeyRightShift:
			println("Done selecting\n")
			keyboard.ShiftKeyIsDown = false
			//foc.Selection.CurrentlySelecting = false // TODO?  possibly flip around if selectionStart comes after selectionEnd in the page flow?

		case msg.KeyLeftControl:
			fallthrough
		case msg.KeyRightControl:
			println("Control RELEASED\n")
			keyboard.ControlKeyIsDown = false

		case msg.KeyLeftAlt:
			fallthrough
		case msg.KeyRightAlt:
			println("Alt RELEASED\n")
			keyboard.AltKeyIsDown = false

		case msg.KeyLeftSuper:
			fallthrough
		case msg.KeyRightSuper:
			println("'Super' modifier key RELEASED\n")
			keyboard.SuperKeyIsDown = false
		}
	} else { //     .Press   or   .Repeat
		//mods
		switch msg.ModifierKey(m.Mod) {
		case msg.ModShift:
			println("PRESSED/REPEATED ModShift --- Started selection\n")
			keyboard.ShiftKeyIsDown = true
			// foc.Selection.CurrentlySelecting = true
			// foc.Selection.StartX = foc.CursX
			// foc.Selection.StartY = foc.CursY
		case msg.ModAlt:
			println("PRESSED/REPEATED ModAlt\n")
			keyboard.AltKeyIsDown = true
		case msg.ModControl:
			println("PRESSED/REPEATED ModControl\n")
			keyboard.ControlKeyIsDown = true
		case msg.ModSuper:
			println("PRESSED/REPEATED ModSuper\n")
			keyboard.SuperKeyIsDown = true
		}

		//keys
		switch m.Key {

		case msg.KeyLeftAlt:
			fallthrough
		case msg.KeyRightAlt:
			t.Terms.Defocus()
			gl.SetArrowPointer()

			/*
				case glfw.KeyEnter:
					startOfLine := b[foc.CursY][:foc.CursX]
					restOfLine := b[foc.CursY][foc.CursX:len(b[foc.CursY])]
					b[foc.CursY] = startOfLine
					b = insert(b, foc.CursY+1, restOfLine)

					foc.CursX = 0
					foc.CursY++
					foc.TextBodies[0] = b

					if foc.CursY >= len(b) {
						foc.CursY = len(b) - 1
					}
				case glfw.KeyHome:
					if eitherControlKeyHeld() {
						foc.CursY = 0
					}

					foc.CursX = 0
					movedCursorSoUpdateDependents()
				case glfw.KeyEnd:
					if eitherControlKeyHeld() {
						foc.CursY = len(b) - 1
					}

					foc.CursX = len(b[foc.CursY])
					movedCursorSoUpdateDependents()
				case glfw.KeyUp:
					if foc.CursY > 0 {
						foc.CursY--

						if foc.CursX > len(b[foc.CursY]) {
							foc.CursX = len(b[foc.CursY])
						}
					}

					movedCursorSoUpdateDependents()
				case glfw.KeyDown:
					if foc.CursY < len(b)-1 {
						foc.CursY++

						if foc.CursX > len(b[foc.CursY]) {
							foc.CursX = len(b[foc.CursY])
						}
					}

					movedCursorSoUpdateDependents()
				case glfw.KeyLeft:
					if foc.CursX == 0 {
						if foc.CursY > 0 {
							foc.CursY--
							foc.CursX = len(b[foc.CursY])
						}
					} else {
						if glfw.ModifierKey(m.Mod) == glfw.ModControl {
							foc.CursX = getWordSkipPos(foc.CursX, -1)
						} else {
							foc.CursX--
						}
					}

					movedCursorSoUpdateDependents()
				case glfw.KeyRight:
					if foc.CursX < len(b[foc.CursY]) {
						if glfw.ModifierKey(m.Mod) == glfw.ModControl {
							foc.CursX = getWordSkipPos(foc.CursX, 1)
						} else {
							foc.CursX++
						}
					}

					movedCursorSoUpdateDependents()
				case glfw.KeyBackspace:
					if foc.CursX == 0 {
						b = remove(b, foc.CursY, b[foc.CursY])
						foc.TextBodies[0] = b
						foc.CursY--
						foc.CursX = len(b[foc.CursY])

					} else {
						foc.RemoveCharacter(false)
					}

				case glfw.KeyDelete:
					foc.RemoveCharacter(true)
					fmt.Println("Key Deleted")
			*/

		}

		//script.Process(false)
	}
}

// func getWordSkipPos(xIn int, change int) int {

// 	peekPos := xIn
// 	foc := Focused
// 	b := foc.TextBodies[0]

// 	for {
// 		peekPos += change

// 		if peekPos < 0 {
// 			return 0
// 		}

// 		if peekPos >= len(b[foc.CursY]) {
// 			return len(b[foc.CursY])
// 		}

// 		if string(b[foc.CursY][peekPos]) == " " {
// 			return peekPos
// 		}
// 	}
// }

// only key/char events need this autoscrolling (to keep cursor visible).
// because mouse can only click visible spots
func movedCursorSoUpdateDependents() {
	/*
		foc := Focused

		// autoscroll to keep cursor visible
		ls := float32(foc.CursX) * gl.CharWid // left side (of cursor, in virtual space)
		rs := ls + gl.CharWid                 // right side ^

		if ls < foc.BarHori.ScrollDelta {
			foc.BarHori.ScrollDelta = ls
		}

		if rs > foc.BarHori.ScrollDelta+foc.Content.Width() {
			foc.BarHori.ScrollDelta = rs - foc.Content.Width()
		}

		// --- Selection Marking ---
		//
		// when SM is made functional,
		// we should probably detect whether cursor
		// position should update Start_ or End_ at this point.
		// rather than always making that the "end".
		// i doubt marking forwards or backwards will ever alter what is
		// done with the selection

		if foc.Selection.CurrentlySelecting {
			foc.Selection.EndX = foc.CursX
			foc.Selection.EndY = foc.CursY
		} else { // moving cursor without shift gets rid of selection
			foc.Selection.StartX = math.MaxUint32
			foc.Selection.StartY = math.MaxUint32
			foc.Selection.EndX = math.MaxUint32
			foc.Selection.EndY = math.MaxUint32
		}
	*/
}
