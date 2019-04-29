package router

import (
	"fmt"
	"io"
)

func Example_makeSetupHandlers() {

	var (
		r  *router
		am ProcManager
		rw io.ReadWriter
	)

	sh, err := makeSetupHandlers(r, am, rw)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("%v\n", sh)

	//Output: ZZZZ
}
