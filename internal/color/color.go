package color

import (
	fatihcolor "github.com/fatih/color"
)

type Color string

const (
	Black Color = "black"
	Red   Color = "red"
	Green Color = "green"
	Blue  Color = "blue"
)

func (c Color) ColorizeText(s string) string {
	switch c {
	case Black:
		return fatihcolor.BlackString("%s", s)
	case Red:
		return fatihcolor.RedString("%s", s)
	case Green:
		return fatihcolor.GreenString("%s", s)
	case Blue:
		return fatihcolor.BlueString("%s", s)
	default:
		return fatihcolor.BlackString("%s", s)
	}
}
