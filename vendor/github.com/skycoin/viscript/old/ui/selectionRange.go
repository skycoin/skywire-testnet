package ui

import (
	//"fmt"
	"math"
)

type SelectionRange struct {
	/*             future considerations/fixme:

	* allow a range (w/ mouse OR arrow keys)
	* need to sanitize start/end positions.
	// since they may be beyond the last line character of the line.
	// also, in addition to backspace/delete, typing any visible character should delete marked text.
	// complication:   if they start or end on invalid characters (of the line string),
	// the forward or backwards direction from there determines where the visible selection
	// range starts/ends....
	// will an end position always be defined (when value is NOT math.MaxUint32),
	// when a START is?  because that determines where the first VISIBLY marked
	// character starts
	*/

	StartX             int
	StartY             int
	EndX               int
	EndY               int
	CurrentlySelecting bool
}

func (sr *SelectionRange) Init() {
	sr.StartX = math.MaxUint32
	sr.StartY = math.MaxUint32
	sr.EndX = math.MaxUint32
	sr.EndY = math.MaxUint32
}
