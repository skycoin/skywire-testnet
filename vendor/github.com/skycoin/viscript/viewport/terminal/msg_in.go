package terminal

import (
	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/msg"
)

func (t *Terminal) UnpackMessage(message []byte) []byte {
	//TODO/FIXME:   cache channel id wherever it may be needed
	message = message[4:] //...for now, DISCARD the channel id prefix

	switch msg.GetType(message) {

	case msg.TypeClear:
		t.clear()
		t.Curr.Y = 0

	case msg.TypeCommand:
		var m msg.MessageCommand
		msg.MustDeserialize(message, &m)
		Terms.OnUserCommand(t.TerminalId, m)

	case msg.TypeCommandLine:
		var m msg.MessageCommandLine
		msg.MustDeserialize(message, &m)
		t.updateCommandLine(m)

	case msg.TypeSetCharAt:
		var m msg.MessageSetCharAt
		msg.MustDeserialize(message, &m)
		t.SetCharacterAt(int(m.X), int(m.Y), m.Char)

	case msg.TypePutChar:
		var m msg.MessagePutChar
		msg.MustDeserialize(message, &m)
		t.PutCharacter(m.Char)

	//lower level messages
	case msg.TypeKey:
		var m msg.MessageKey
		msg.MustDeserialize(message, &m)
		t.onKey(m)

	case msg.TypeMouseScroll:
		var m msg.MessageMouseScroll
		msg.MustDeserialize(message, &m)
		t.onMouseScroll(m)

	default:
		app.At("viewport/terminal/msg_in", "*********** UNHANDLED MESSAGE TYPE! ***********")

	}

	return message
}

//
//EVENT HANDLERS
//

func (t *Terminal) onKey(m msg.MessageKey) {
	switch m.Key {
	case msg.KeyEnter:
		t.NewLine()
	}
}

func (t *Terminal) onMouseScroll(m msg.MessageMouseScroll) {
	if m.HoldingControl {
		//only using m.Y because
		//m.X is sideways scrolling (which most mice can't do)
		y := float32(m.Y)
		changeFactor := float32(1 + app.Clamp(y, -1, 1)/10)
		newWidth := t.Bounds.Width() * changeFactor
		newHeight := t.Bounds.Height() * changeFactor

		if newWidth < 0.2 ||
			newHeight < 0.2 {
			return
		}

		t.Bounds.Right = t.Bounds.Left + newWidth
		t.Bounds.Bottom = t.Bounds.Top - newHeight
		t.CharSize.X *= changeFactor
		t.CharSize.Y *= changeFactor
	}
}
