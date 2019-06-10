package ioutil

import "sync/atomic"

// AtomicBool implements a thread-safe boolean value.
type AtomicBool struct {
	flag int32
}

// Set set's the boolean to specified value
// and returns true if the value is changed.
func (b *AtomicBool) Set(v bool) bool {
	newF := int32(0)
	if v {
		newF = 1
	}
	return newF != atomic.SwapInt32(&b.flag, newF)
}

// Get obtains the current boolean value.
func (b *AtomicBool) Get() bool {
	return atomic.LoadInt32(&b.flag) == 1
}
