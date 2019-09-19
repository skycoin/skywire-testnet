package ioutil

import (
	"bytes"
)

// BufRead is designed to help writing 'io.Reader' implementations.
// It reads from 'data' into 'p'. If 'p' is short, write to 'buf'.
// Note that one should check if 'buf' has data and read from that first before calling this function.
func BufRead(buf *bytes.Buffer, data, p []byte) (int, error) {
	n := copy(p, data)
	if n < len(data) {
		if _, err := buf.Write(data[n:]); err != nil {
			log.WithError(err).Warn("Failed to write to buffer")
		}
	}
	return n, nil
}
