// Package testhelpers provides helpers for testing.
package testhelpers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const timeout = 5 * time.Second

// WithinTimeout tries to read an error from error channel within timeout and returns it.
// If timeout exceeds, nil value is returned.
func WithinTimeout(ch <-chan error) error {
	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		return nil
	}
}

// NoErrorN performs require.NoError on multiple errors
func NoErrorN(t *testing.T, errs ...error) {
	for _, err := range errs {
		require.NoError(t, err)
	}
}
