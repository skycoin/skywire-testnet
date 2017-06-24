package ui

import (
	"github.com/skycoin/viscript/app"
)

type Button struct {
	Name      string
	Activated bool
	Rect      *app.PicRectangle
}
