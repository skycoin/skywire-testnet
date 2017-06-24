package app

/*

This package's functionality is often called "common" or "util", but
I think it reads cleaner with this name.
However, newcomers COULD confuse it with "package viewport"?

*/

const Name = "V I S C R I P T"
const UvSpan = float32(1.0) / 16 //span of a tile/cell in uv space

func At(path, s string) { //report the func location of currently running code
	//centering assumes 80 columns

	//center location
	lp := (78 - len(path)) / 2 //location padding (considers angle bracket wrap)
	bar := ""

	for i := 0; i < lp; i++ {
		bar += "-"
	}

	println(bar + " <" + path + ">")

	//center func
	lp = (78 - len(s)) / 2 //func padding (considers parens)
	bar = ""

	for i := 0; i < lp; i++ {
		bar += "_"
	}

	println(bar + s + "()")
}

// params: float value, negativemost, & positivemost bounds
func Clamp(f, negBoundary, posBoundary float32) float32 {
	if f < negBoundary {
		f = negBoundary
	}

	if f > posBoundary {
		f = posBoundary
	}

	return f
}

// params: Rectangle, negativemost, & positivemost bounds
func ClampLeftAndRightOf(r *Rectangle, negBoundary, posBoundary float32) *Rectangle {
	if r.Left < negBoundary {
		r.Left = negBoundary
	}
	if r.Right > posBoundary {
		r.Right = posBoundary
	}

	return r
}

// WARNING: given arguments must be in range
func Insert(slice []string, index int, value string) []string {
	slice = slice[0 : len(slice)+1]      // grow the slice by one element
	copy(slice[index+1:], slice[index:]) // move the upper part of the slice out of the way and open a hole
	slice[index] = value
	return slice
}

// i believe JUSTIN created this, when he attempted to implement autoscroll
// ("similar to insert method, instead moves current slice element and appends to one above")
func Remove(slice []string, index int, value string) []string {
	slice = append(slice[:index], slice[index+1:]...)
	slice[index-1] = slice[index-1] + value
	return slice
}
