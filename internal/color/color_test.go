package color

import (
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

const (
	escape = "\x1b"
	reset  = escape + "[0m"
	red    = escape + "[31m"
	green  = escape + "[32m"
	blue   = escape + "[34m"
	black  = escape + "[30m"
)

func TestColorizeText(t *testing.T) {
	cases := []struct {
		name   string
		color  Color
		text   string
		result string
	}{
		{
			name:   "red color",
			color:  Red,
			text:   "red text",
			result: red + "red text" + reset,
		},
		{
			name:   "green color",
			color:  Green,
			text:   "green text",
			result: green + "green text" + reset,
		},
		{
			name:   "blue color",
			color:  Blue,
			text:   "blue text",
			result: blue + "blue text" + reset,
		},
		{
			name:   "black color",
			color:  Black,
			text:   "black text",
			result: black + "black text" + reset,
		},
		{
			name:   "bad value",
			color:  "BAD_VALUE",
			text:   "black text",
			result: black + "black text" + reset,
		},
	}

	color.NoColor = false

	for _, tc := range cases {
		result := tc.color.ColorizeText(tc.text)
		assert.Equal(t, result, tc.result)
	}
}
