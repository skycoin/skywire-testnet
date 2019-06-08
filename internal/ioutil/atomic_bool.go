package ioutil

import "sync/atomic"

type AtomicBool struct {
	flag int32
}

func (b *AtomicBool) Set(v bool) bool {
	newF := int32(0)
	if v {
		newF = 1
	}
	return newF != atomic.SwapInt32(&b.flag, newF)
}

func (b *AtomicBool) Get() bool {
	return atomic.LoadInt32(&b.flag) == 1
}
