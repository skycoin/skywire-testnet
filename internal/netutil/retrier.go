package netutil

import (
	"errors"
	"github.com/prometheus/common/log"
	"net"
	"time"
)

var ErrThresholdReached = errors.New("threshold timeout has been reached")

type ConnFunc func(conn *net.Conn) error

type Retrier struct {
	exponentialBackoff time.Duration
	exponentialFactor uint32
	threshold time.Duration
	errWhitelist map[error]struct{}
}

func NewRetrier(exponentialBackoff, threshold time.Duration, factor uint32) *Retrier {
	return &Retrier{
		exponentialBackoff: exponentialBackoff,
		threshold: threshold,
		exponentialFactor: factor,
		errWhitelist:make(map[error]struct{}),
	}
}

func (r *Retrier) WithErrWhitelist(errors ...error) *Retrier {
	m := make(map[error]struct{})
	for _, err := range errors {
		m[err] = struct{}{}
	}

	r.errWhitelist = m
	return r
}

func (r Retrier) Do(conn *net.Conn, f ConnFunc) error {
	counter := time.Duration(0)
	var err error

	for counter < r.threshold {
		err = f(conn)
		if err == nil {
			return nil
		}

		if r.isWhitelisted(err) {
			return err
		}

		log.Warn(err)
		counter += time.Duration(r.exponentialFactor) * r.exponentialBackoff
	}

	return ErrThresholdReached
}

func (r Retrier) isWhitelisted(err error) bool {
	_, ok := r.errWhitelist[err]
	return ok
}