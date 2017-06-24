package script

/*
	"fmt"
	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/gfx"
	"github.com/skycoin/viscript/tree"
	"strings"
	//"github.com/skycoin/viscript/ui"
	"math"
	"regexp"
	"strconv"
*/

/*

KNOWN ISSUES:

* only looks for 1 expression per line

* no attempt is made to get anything inside any block
	which may come after the opening curly brace, on the same line

* closing curly brace of a block only recognized as a "}" amidst spaces



TODO:

* make sure identifiers are unique within a partic scope

*/

// lexing

/*
var integralTypes = []string{"bool", "int32", "float32", "rune", "string"} // FIXME: should allow [] and [42] prefixes
var integralFuncs = []string{"add32", "sub32", "mult32", "div32"}
var operators = []string{"=", ":=", "==", "!=", "+=", "-=", "*=", "/=", "%=", "+", "-", "*", "/", "%"}
var keywords = []string{"break", "continue", "fallthrough", "var", "if", "do", "while", "switch", "case", "for", "func", "return", "range"}
var varIdentifiers = []string{}
var funcIdentifiers = []string{}
var tokens = []*Token{}
var mainBlock = &CodeBlock{Name: "main"} // the root/entry/top/alpha level of the program
var currBlock = mainBlock
var parenCaptureTier = 0 // number of levels of parenception
var textColors = []*gfx.ColorSpot{}

// REGEX scanning (raw strings to avoid having to quote backslashes)
var declaredVar = regexp.MustCompile(`^( +)?var( +)?([a-zA-Z]\w*)( +)?int32(( +)?=( +)?([0-9]+))?$`)
var declFuncStart = regexp.MustCompile(`^func ([a-zA-Z]\w*)( +)?\((.*)\)( +)?\{$`)
var declFuncEnd = regexp.MustCompile(`^( +)?\}( +)?$`)
var calledFunc = regexp.MustCompile(`^( +)?([a-zA-Z]\w*)\(([0-9]+|[a-zA-Z]\w*),( +)?([0-9]+|[a-zA-Z]\w*)\)$`)

const (
	LexType_Keyword = iota
	LexType_Operator
	LexType_IntegralType
	LexType_IntegralFunc
	// literals
	LexType_LiteralInt32
	LexType_LiteralFloat32
	LexType_LiteralRune
	LexType_LiteralString
	// delimiters
	LexType_ParenStart
	LexType_ParenEnd
	LexType_BlockStart
	LexType_BlockEnd
	LexType_RuneLiteralStart
	LexType_RuneLiteralEnd
	LexType_StringLiteralStart
	LexType_StringLiteralEnd
	// user identifiers
	LexType_IdentifierVar  // user variable
	LexType_IdentifierFunc // user func
)

// ^^^
// as above, so below   (keep these synchronized)
// VVV

func lexTypeString(i int) string {
	switch i {
	case LexType_Keyword:
		return "Keyword"
	case LexType_Operator:
		return "Operator"
	case LexType_IntegralType:
		return "IntegralType"
	case LexType_IntegralFunc:
		return "IntegralFunc"

	// literals
	case LexType_LiteralInt32:
		return "LiteralInt32"
	case LexType_LiteralFloat32:
		return "LiteralFloat32"
	case LexType_LiteralRune:
		return "LiteralRune"
	case LexType_LiteralString:
		return "LiteralString"

	// delimiters
	case LexType_ParenStart:
		return "ParenStart"
	case LexType_ParenEnd:
		return "ParenEnd"
	case LexType_BlockStart:
		return "BlockStart"
	case LexType_BlockEnd:
		return "BlockEnd"
	case LexType_RuneLiteralStart:
		return "RuneLiteralStart"
	case LexType_RuneLiteralEnd:
		return "RuneLiteralEnd"
	case LexType_StringLiteralStart:
		return "StringLiteralStart"
	case LexType_StringLiteralEnd:
		return "StringLiteralEnd"

	// user idents
	case LexType_IdentifierVar:
		return "IdentifierVar"
	case LexType_IdentifierFunc:
		return "IdentifierFunc"

	default:
		return "ERROR! LexType case unknown!"
	}
}

func MakeTree() {
	// setup hardwired test tree
	tI := len(Terms) // terminal id

	// new panel
	Terms = append(Terms, &gfx.Terminal{FractionOfStrip: 1})
	Terms[tI].Init()

	// new tree
	Terms[tI].Trees = append(
		Terms[tI].Trees, &tree.Tree{tI, []*tree.Node{}})
	//makeNode(tI, 1, math.MaxInt32, math.MaxInt32, "top") // 0
	makeNode(tI, 1, 2, math.MaxInt32, "top")                  // 0
	makeNode(tI, 6, math.MaxInt32, 0, "1st left")             // 1
	makeNode(tI, math.MaxInt32, 3, 0, "1st right")            // 2
	makeNode(tI, 4, math.MaxInt32, 2, "level 3")              // 3
	makeNode(tI, math.MaxInt32, 5, 3, "level 4")              // 4
	makeNode(tI, math.MaxInt32, math.MaxInt32, 4, "level 5")  // 5
	makeNode(tI, math.MaxInt32, math.MaxInt32, 1, "freddled") // 6
}

func makeNode(panelId, childIdL, childIdR, parentId int, s string) {
	Terms[panelId].Trees[0].Nodes = append(
		Terms[panelId].Trees[0].Nodes, &tree.Node{s, childIdL, childIdR, parentId})
}

func Process(feedbackWanted bool) {
	// clear script
	mainBlock = &CodeBlock{Name: "main"}

	// clear OS and graphical consoles
	app.Con.Lines = []string{}
	Terms[1].TextBodies[0] = []string{}

	if feedbackWanted {
		app.MakeHighlyVisibleLogEntry(`LEXING`, 5)
	}
	lexAll()

	//if feedbackWanted {
	//	app.MakeHighlyVisibleLogEntry(`PARSING`, 5)
	//}
	//parseAll()

	if feedbackWanted {
		app.MakeHighlyVisibleLogEntry(`RUNNING`, 5)
	}
	run(mainBlock)
}

func lexAll() {
	bods := Terms[0].TextBodies

	textColors = []*gfx.ColorSpot{}

	for i, line := range bods[0] {
		lexAndColorize(i, line)
	}

	// FIXME when needing colors in non-script terminals
	Terms[0].TextColors = textColors
}

func lexAndColorize(y int, line string) string {
	s := line // the dynamic/processed offshoot string
	comment := ""

	// strip any comments
	x := strings.Index(line, "//")
	if x != -1 {
		// (comment exists)
		s = line[:x]
		comment = line[x:]
	}

	start := 0 // start of non-space runes
	for _, ru := range s {
		if ru == ' ' {
			start++
		} else {
			break
		}
	}

	x = start
	s = strings.TrimSpace(s)

	if  len(s) > 0 {
		// (we're not left with an empty string)
		//fmt.Println("s:", s)

		// tokenize
		lex := strings.Split(s, " ")

		for i := range lex {
			//fmt.Printf("lexer element %d: \"%s\"\n", i, lex[i])

			switch {
			case tokenizedAny(keywords, LexType_Keyword, lex[i]):
				color(x, y, gfx.MaroonDark)
			case tokenizedAny(operators, LexType_Operator, lex[i]):
				color(x, y, gfx.Maroon)
			case tokenizedAny(integralTypes, LexType_IntegralType, lex[i]):
				color(x, y, gfx.Cyan)
			case tokenizedAny(integralFuncs, LexType_IntegralFunc, lex[i]):
				color(x, y, gfx.Maroon)
			case tokenizedAny(varIdentifiers, LexType_IdentifierVar, lex[i]):
				color(x, y, gfx.White)
			case tokenizedAny(funcIdentifiers, LexType_IdentifierFunc, lex[i]):
				color(x, y, gfx.Green)
			default:
				color(x, y, gfx.White)
			}

			x += len(lex[i])

			if i != len(lex)-1 {
				// (not the last token)
				x++ // for a space
			}
		}

		line = strings.Join(lex, " ")
		regexLine(y, s)
	} else {
		line = ""
	}

	if comment != "" {
		line += " " + comment
		color(x, y, gfx.GrayDark)
	}

	return line
}

func color(x, y int, color []float32) {
	textColors = append(textColors, &gfx.ColorSpot{app.Vec2I{x, y}, color})
	//fmt.Println("------------textColors[len(textColors)-1]:", textColors[len(textColors)-1])
}

func tokenizedAny(slice []string, i int, elem string) bool {
	for j := range slice {
		if elem == slice[j] {
			tokens = append(tokens, &Token{i, elem})
			//fmt.Printf("<<<<<<<<<<<<<< TOKENIZED %s >>>>>>>>>>>>>>: %s\n", lexTypeString(i), `"`+elem+`"`)
			return true
		} else { // to allow tokenizing bundled func name and opening paren, in that order & separately
			if i == LexType_IntegralFunc || i == LexType_IdentifierFunc {
				// (looking for func)
				if strings.Index(elem, slice[j]) != -1 {
					// (this contains func name)
					// look for opening of enclosing pairs
					for k, c := range elem {
						switch c {
						case '(':
							//fmt.Println("ENCOUNTERED open paren")
							tokens = append(tokens, &Token{i, elem[:k]})
							//fmt.Printf("<<<<<<<<<<<<<< TOKENIZED %s >>>>>>>>>>>>>>: %s\n", lexTypeString(i), `"`+elem[:k]+`"`)
							tokens = append(tokens, &Token{LexType_ParenStart, "("})
							//fmt.Printf("<<<<<<<<<<<<<< TOKENIZED %s >>>>>>>>>>>>>>: %s\n", lexTypeString(LexType_ParenStart), `"("`)

							parenCaptureTier++

							if len(elem) > k+1 {
								oneOrMoreParams := elem[k+1:]
								fmt.Println("oneOrMoreParams:", oneOrMoreParams)
							}

							return true
						}
					}
				}
			}
		}
	}

	return false
}

func regexLine(i int, line string) {
	// scan for high level pieces
	switch {
	case declaredVar.MatchString(line):
		result := declaredVar.FindStringSubmatch(line)

		var s = fmt.Sprintf("%d: var (%s) declared", i, result[3])
		//printIntsFrom(currBlock)

		if result[8] == "" {
			currBlock.VarInt32s = append(currBlock.VarInt32s, &VarInt32{result[3], 0})
		} else {
			value, err := strconv.Atoi(result[8])
			if err != nil {
				s = fmt.Sprintf("%s... BUT COULDN'T CONVERT ASSIGNMENT (%s) TO A NUMBER!", s, result[8])
			} else {
				currBlock.VarInt32s = append(currBlock.VarInt32s, &VarInt32{result[3], int32(value)})
				s = fmt.Sprintf("%s & assigned: %d", s, value)
			}
		}

		app.Con.Add(fmt.Sprintf("%s\n", s))
	case declFuncStart.MatchString(line):
		result := declFuncStart.FindStringSubmatch(line)

		app.Con.Add(fmt.Sprintf("%d: func (%s) declared, with params: %s\n", i, result[1], result[3]))

		if currBlock.Name == "main" {
			currBlock = &CodeBlock{Name: result[1]}
			mainBlock.SubBlocks = append(mainBlock.SubBlocks, currBlock) // FUTURE FIXME: methods in structs shouldn't be on main/root func
		} else {
			app.Con.Add("Func'y func-ception! CAN'T PUT A FUNC INSIDE A FUNC!\n")
		}
	case declFuncEnd.MatchString(line):
		app.Con.Add(fmt.Sprintf("func close...\n"))
		//printIntsFrom(mainBlock)
		//printIntsFrom(currBlock)

		if currBlock.Name == "main" {
			app.Con.Add(fmt.Sprintf("ERROR! Main\\Root level function doesn't need enclosure!\n"))
		} else {
			currBlock = mainBlock
		}
	case calledFunc.MatchString(line): // FIXME: hardwired for 2 params each
		result := calledFunc.FindStringSubmatch(line)

		app.Con.Add(fmt.Sprintf("%d: func call (%s) expressed\n", i, result[2]))
		app.Con.Add(fmt.Sprintf("currBlock: %s\n", currBlock))
		currBlock.Expressions = append(currBlock.Expressions, line)

		//currBlock.Expressions = append(currBlock.Expressions, result[2])
		//currBlock.Parameters = append(currBlock.Parameters, result[3])
		//currBlock.Parameters = append(currBlock.Parameters, result[5])


		// // prints out all captures
		// for i, v := range result {
		// 	app.Con.Add(fmt.Sprintf("%d. %s\n", i, v))
		// }

	case line == "":
		// just ignore
	default:
		app.Con.Add(fmt.Sprintf("SYNTAX ERROR on line %d: \"%s\"\n", i, line))
	}
}

func run(pb *CodeBlock) { // passed block of code
	app.Con.Add(fmt.Sprintf("running function: '%s'\n", pb.Name))

	for i, line := range pb.Expressions {
		app.Con.Add(fmt.Sprintf("------evaluating expression: '%s\n", line))

		switch {
		case calledFunc.MatchString(line): // FIXME: hardwired for 2 params each
			result := calledFunc.FindStringSubmatch(line)
			app.Con.Add(fmt.Sprintf("%d: calling func (%s) with params: %s, %s\n", i, result[2], result[3], result[5]))

			a := getInt32(result[3])
			if a == math.MaxInt32 {
				// (not legit num)
				return
			}
			b := getInt32(result[5])
			if b == math.MaxInt32 {
				// (not legit num)
				return
			}

			switch result[2] {
			case "add32":
				app.Con.Add(fmt.Sprintf("%d + %d = %d\n", a, b, a+b))
			case "sub32":
				app.Con.Add(fmt.Sprintf("%d - %d = %d\n", a, b, a-b))
			case "mult32":
				app.Con.Add(fmt.Sprintf("%d * %d = %d\n", a, b, a*b))
			case "div32":
				app.Con.Add(fmt.Sprintf("%d / %d = %d\n", a, b, a/b))
			default:
				for _, cb := range pb.SubBlocks {
					app.Con.Add((fmt.Sprintf("CodeBlock.Name considered: %s   switching on: %s\n", cb.Name, result[2])))

					if cb.Name == result[2] {
						app.Con.Add((fmt.Sprintf("'%s' matched '%s'\n", cb.Name, result[2])))
						run(cb)
					}
				}
			}
		}
	}
}

func getInt32(s string) int32 {
	value, err := strconv.Atoi(s)

	if err != nil {
		for _, v := range currBlock.VarInt32s {
			if s == v.Name {
				return v.Value
			}
		}

		if currBlock.Name != "main" {
			for _, v := range mainBlock.VarInt32s {
				if s == v.Name {
					return v.Value
				}
			}
		}

		app.Con.Add(fmt.Sprintf("ERROR!  '%s' IS NOT A VALID VARIABLE/FUNCTION!\n", s))
		return math.MaxInt32
	}

	return int32(value)
}

func printIntsFrom(f *CodeBlock) {
	if len(f.VarInt32s) == 0 {
		app.Con.Add(fmt.Sprintf("%s has no elements!\n", f.Name))
	} else {
		for i, v := range f.VarInt32s {
			app.Con.Add(fmt.Sprintf("%s.VarInt32s[%d]: %s = %d\n", f.Name, i, v.Name, v.Value))
		}
	}
}
*/

/*
The FindAllStringSubmatch-function will, for each match, return an array with the
entire match in the first field and the
content of the groups in the remaining fields.
The arrays for all the matches are then captured in a container array.

the number of fields in the resulting array always matches the number of groups plus one.
*/
