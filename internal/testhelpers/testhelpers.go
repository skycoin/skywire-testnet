// Package testhelpers provides helpers for testing.
package testhelpers

import (
	"time"
)

// Timeout defines timeout for NoErrorWithinTimeout
var Timeout = 5 * time.Second

// joinErrChannels multiplexes several error channels
func joinErrChannels(errChans []<-chan error) chan error {
	joinedCh := make(chan error)
	for _, ch := range errChans {
		go func(errCh <-chan error) {
			joinedCh <- <-errCh
		}(ch)
	}
	return joinedCh
}

// NoErrorWithinTimeout tries to read an error from error channel within timeout and returns it.
// If timeout exceeds, nil value is returned.
func NoErrorWithinTimeout(ch <-chan error) error {
	select {
	case err := <-ch:
		return err
	case <-time.After(Timeout):
		return nil
	}
}

// NoErrorWithinTimeoutN tries to read an error from error channels within timeout and returns it.
// If timeout exceeds, nil value is returned.
func NoErrorWithinTimeoutN(errChans ...<-chan error) error {
	return NoErrorWithinTimeout(joinErrChannels(errChans))
}
