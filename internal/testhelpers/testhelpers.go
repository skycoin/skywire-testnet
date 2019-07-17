// Package testhelpers provides helpers for testing.
package testhelpers

import (
	"time"
)

const timeout = 5 * time.Second

// NoErrorWithinTimeout tries to read an error from error channel within timeout and returns it.
// If timeout exceeds, nil value is returned.
func NoErrorWithinTimeout(ch <-chan error) error {
	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		return nil
	}
}
