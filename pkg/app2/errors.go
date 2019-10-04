package app2

import (
	"errors"
)

var (
	// ErrPortAlreadyBound is being returned when trying to bind to the port
	// which is already bound to.
	ErrPortAlreadyBound = errors.New("port is already bound")
)

var (
	// errMethodNotImplemented serves as a return value for non-implemented funcs (stubs).
	errMethodNotImplemented = errors.New("method not implemented")
)
