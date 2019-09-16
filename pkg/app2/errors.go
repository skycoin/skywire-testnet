package app2

import "github.com/pkg/errors"

var (
	ErrPortAlreadyBound = errors.New("port is already bound")
)
