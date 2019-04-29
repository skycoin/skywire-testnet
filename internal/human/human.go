package human

import (
	"context"

	"github.com/skycoin/skywire/internal/color"
)

// UserOption is an option given to a user.
type UserOption struct {
	Label string      `json:"label"`
	Color color.Color `json:"color"`
}

// InputGrabber is responsible for obtaining user input.
type InputGrabber interface {
	// GrabInput obtains the users input.
	GrabInput(ctx context.Context, msg, details string, opts []UserOption) (string, error)
}
