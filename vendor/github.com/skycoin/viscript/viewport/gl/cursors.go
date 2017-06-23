package gl

import (
	"time"

	"github.com/skycoin/viscript/app"
)

var Curs Cursors = Cursors{nextFrame: time.Now()}

type Cursors struct {
	// private
	nextFrame      time.Time
	shrinking      bool
	shrinkFraction float32
}

func (c *Cursors) Tick() {
	var speedFactor float32 = 0.06

	if c.nextFrame.Before(time.Now()) {
		c.nextFrame = time.Now().Add(time.Millisecond * 16) // 170 was for simple on/off blinking

		if c.shrinking {
			c.shrinkFraction -= speedFactor

			if c.shrinkFraction < 0.2 {
				c.shrinking = false
			}
		} else {
			c.shrinkFraction += speedFactor

			if c.shrinkFraction > 0.8 {
				c.shrinking = true
			}
		}
	}
}

func (c *Cursors) GetCurrentFrame(r app.Rectangle) *app.Rectangle {
	if c.shrinking {
		r.Bottom = r.Top - c.shrinkFraction*r.Height()
		r.Left = r.Right - c.shrinkFraction*r.Width()
	} else { // growing
		r.Top = r.Bottom + c.shrinkFraction*r.Height()
		r.Right = r.Left + c.shrinkFraction*r.Width()
	}

	return &r
}
