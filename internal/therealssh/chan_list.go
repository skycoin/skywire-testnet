package therealssh

import "sync"

type chanList struct {
	sync.Mutex

	chans []*SSHChannel
}

func newChanList() *chanList {
	return &chanList{chans: []*SSHChannel{}}
}

func (c *chanList) add(sshCh *SSHChannel) uint32 {
	c.Lock()
	defer c.Unlock()

	for i := range c.chans {
		if c.chans[i] == nil {
			c.chans[i] = sshCh
			return uint32(i)
		}
	}

	c.chans = append(c.chans, sshCh)
	return uint32(len(c.chans) - 1)
}

func (c *chanList) getChannel(id uint32) *SSHChannel {
	c.Lock()
	defer c.Unlock()

	if id < uint32(len(c.chans)) {
		return c.chans[id]
	}

	return nil
}

func (c *chanList) dropAll() []*SSHChannel {
	c.Lock()
	defer c.Unlock()
	var r []*SSHChannel

	for _, ch := range c.chans {
		if ch == nil {
			continue
		}
		r = append(r, ch)
	}
	c.chans = nil
	return r
}
