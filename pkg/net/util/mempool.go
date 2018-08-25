package util

import "sync"

var (
	FixedMtuPool = NewFixedSizePool(1500)
)

type FixedSizePool struct {
	pool sync.Pool
	Size int
}

func NewFixedSizePool(size int) (fp *FixedSizePool) {
	fp = &FixedSizePool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
		Size: size,
	}
	return fp
}

func (fp *FixedSizePool) Get() []byte {
	v := fp.pool.Get()
	return v.([]byte)
}

func (fp *FixedSizePool) Put(c []byte) {
	if len(c) != fp.Size {
		if cap(c) != fp.Size {
			return
		}
		c = c[:fp.Size]
	}
	fp.pool.Put(c)
}
