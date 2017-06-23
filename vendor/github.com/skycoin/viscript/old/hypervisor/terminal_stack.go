package hypervisor

import (
	"github.com/skycoin/viscript/msg"
)

func onMouseScroll(m msg.MessageMouseScroll) {
	/*
		var delta float32 = 30

		if eitherControlKeyHeld() { // horizontal ability from 1D scrolling
			ScrollTermThatHasMousePointer(float32(m.Y)*-delta, 0)
		} else { // can handle both x & y for 2D scrolling
			ScrollTermThatHasMousePointer(float32(m.X)*delta, float32(m.Y)*-delta)
		}
	*/
}

// func ScrollTermThatHasMousePointer(mousePixelDeltaX, mousePixelDeltaY float32) {
// 	for _, t := range Terms {
// 		t.ScrollIfMouseOver(mousePixelDeltaX, mousePixelDeltaY)
// 	}
// }

// func InsertRuneIntoDocument(s string, message uint32) string {
// 	f := Focused
// 	b := f.TextBodies[0]
// 	resultsDif := f.CursX - len(b[f.CursY])
// 	fmt.Printf("Rune   [%s: %s]", s, string(message))

// 	if f.CursX > len(b[f.CursY]) {
// 		b[f.CursY] = b[f.CursY][:f.CursX-resultsDif] + b[f.CursY][:len(b[f.CursY])] + string(message)
// 		fmt.Printf("line is %s\n", b[f.CursY])
// 		f.CursX++
// 	} else {
// 		b[f.CursY] = b[f.CursY][:f.CursX] + string(message) + b[f.CursY][f.CursX:len(b[f.CursY])]
// 		f.CursX++
// 	}

// 	return string(message)
// }
