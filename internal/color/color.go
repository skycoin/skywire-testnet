package color

import (
	fatihcolor "github.com/fatih/color"
)

// Color is a color that the console output may be colorized to.
type Color string

const (
	// Black color.
	Black Color = "black"
	// Red color.
	Red Color = "red"
	// Green color.
	Green Color = "green"
	// Blue color.
	Blue Color = "blue"
)

// ColorizeText colorizes the passed string and returns the result.
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
