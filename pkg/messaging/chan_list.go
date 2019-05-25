package messaging

import "sync"

type chanList struct {
	sync.Mutex

	chans map[byte]*msgChannel
}

func newChanList() *chanList {
	return &chanList{chans: map[byte]*msgChannel{}}
}

func (c *chanList) add(mCh *msgChannel) byte {
	c.Lock()
	defer c.Unlock()

	for i := byte(0); i < 255; i++ {
		if c.chans[i] == nil {
			c.chans[i] = mCh
			return i
		}
	}

	panic("no free channels")
}

func (c *chanList) get(id byte) *msgChannel {
	c.Lock()
	ch := c.chans[id]
	c.Unlock()

	return ch
}

func (c *chanList) remove(id byte) {
	c.Lock()
	delete(c.chans, id)
	c.Unlock()
}

func (c *chanList) dropAll() []*msgChannel {
	c.Lock()
	defer c.Unlock()
	var r []*msgChannel

	for _, ch := range c.chans {
		if ch == nil {
			continue
		}
		r = append(r, ch)
	}
	c.chans = nil
	return r
}
