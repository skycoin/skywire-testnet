package ioutil

import (
	"bytes"
	"fmt"
	"io"
)

// BufRead is designed to help writing 'io.Reader' implementations.
// It reads from 'data' into 'p'. If 'p' is short, write to 'buf'.
// Note that one should check if 'buf' has data and read from that first before calling this function.
func BufRead(buf bytes.Buffer, data, p []byte) (n int, err error) {
	fmt.Printf("skywire_ioutil: data_len(%d) p_len(%d)\n", len(data), len(p))

	if len(data) > len(p) {
		fmt.Println("skywire_ioutil: writing to buffer.")
		if _, err := buf.Write(data[len(p):]); err != nil {
			fmt.Printf("skywire_ioutil: n(%d) err(%v)\n", 0, io.ErrShortBuffer)
			return 0, io.ErrShortBuffer
		}
	}
	n = copy(p, data)
	fmt.Printf("skywire_ioutil: n(%d) err(%v)\n", n, err)
	return n, nil
}