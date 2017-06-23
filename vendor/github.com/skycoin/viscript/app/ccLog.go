package app

import (
	"fmt"
	//"github.com/skycoin/viscript/hypervisor"
	"strings"
)

var Con = CcLog{} //console log, displays runtime feedback
//private
var fillChar = "*"        //character used to surround/highlight text
const numUsableChars = 79 //assumes 80 column lines.  but to still look fine
//....with more columns, we make each a new line & reserve the last column

type CcLog struct {
	Name  string
	Lines []string
}

func (log CcLog) Add(s string) {
	fmt.Printf(s)
	s = strings.Replace(s, "\n", "", -1)
	log.Lines = append(log.Lines, s)

	/*
		if len(Terms) > 1 {
			Terms[1].TextBodies[0] = append(Terms[1].TextBodies[0], s)
		}
	*/
}

func GetBarOfChars(char string, howMany int) string {
	bar := ""

	for i := 0; i < howMany; i++ {
		bar += char
	}

	return bar
}

// numLines: use odd number for an exact middle point
func MakeHighlyVisibleLogEntry(s string, numLines int) {
	s = " " + s + " "
	osOnly := s == Name

	bar := GetBarOfChars(fillChar, numUsableChars)

	var spaces string
	for i := 0; i < len(s); i++ {
		spaces += " "
	}

	var bookend string
	for i := 0; i < (numUsableChars-len(s))/2; i++ {
		bookend += fillChar
	}

	vMid := numLines / 2 //vertical middle
	for i := 0; i < numLines; i++ {
		switch {
		case i == vMid:
			predPrint(osOnly, bookend+s+bookend)
		case i == vMid-1 || i == vMid+1:
			predPrint(osOnly, bookend+spaces+bookend)
		default:
			predPrint(osOnly, bar)
		}
	}
}

// prints only to OS console window if it's for the app's name
func predPrint(osOnly bool, s string) {
	if len(s) < numUsableChars {
		s += fillChar
	}

	if osOnly {
		println(s)
	} else {
		Con.Add(fmt.Sprintf("%s\n", s))
	}
}
