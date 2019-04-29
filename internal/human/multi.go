package human

import (
	"context"
)

// MultiGrabber combines multiple InputGrabber implementations,
// where the first user response of every request is accepted.
type MultiGrabber struct {
	grabbers []InputGrabber
}

// NewMultiGrabber creates a new MultiGrabber.
func NewMultiGrabber(grabbers ...InputGrabber) InputGrabber {
	return &MultiGrabber{
		grabbers: grabbers,
	}
}

type multiGrabInputResult struct {
	result string
	err    error
}

// GrabInput obtains the users input.
func (g *MultiGrabber) GrabInput(ctx context.Context, msg, details string, opts []UserOption) (string, error) {
	resultCh := make(chan multiGrabInputResult, len(g.grabbers))

	for _, grabber := range g.grabbers {
		go func() {
			res, err := grabber.GrabInput(ctx, msg, details, opts)
			resultCh <- multiGrabInputResult{
				result: res,
				err:    err,
			}
		}()
	}

	firstResult := <-resultCh
	return firstResult.result, firstResult.err
}
