package netutil

import (
	"errors"
	"time"

	"github.com/prometheus/common/log"
)

// Package errors
var (
	ErrMaximumRetriesReached = errors.New("maximum retries attempted without success")
)

// RetryFunc is a function used as argument of (*Retrier).Do(), which will retry on error unless it is whitelisted
type RetryFunc func() error

// Retrier holds a configuration for how retries should be performed
type Retrier struct {
	exponentialBackoff time.Duration // multiplied on every retry by a exponentialFactor
	exponentialFactor  uint32        // multiplier for the backoff duration that is applied on every retry
	times              uint32        // number of times that the given function is going to be retried until success, if 0 it will be retried forever until success
	errWhitelist       map[error]struct{}
}

// NewRetrier returns a retrier that is ready to call Do() method
func NewRetrier(exponentialBackoff time.Duration, times, factor uint32) *Retrier {
	return &Retrier{
		exponentialBackoff: exponentialBackoff,
		times:              times,
		exponentialFactor:  factor,
		errWhitelist:       make(map[error]struct{}),
	}
}

// WithErrWhitelist sets a list of errors into the retrier, if the RetryFunc provided to Do() fails with one of them it will return inmediatelly with such error
func (r *Retrier) WithErrWhitelist(errors ...error) *Retrier {
	m := make(map[error]struct{})
	for _, err := range errors {
		m[err] = struct{}{}
	}

	r.errWhitelist = m
	return r
}

// Do takes a RetryFunc and attempts to execute it, if it fails with an error it will be retried a maximum of given times with an exponentialBackoff, until it returns
// nil or an error that is whitelisted
func (r Retrier) Do(f RetryFunc) error {
	if r.times == 0 {
		return r.retryUntilSuccess(f)
	}

	return r.retryNTimes(f)
}

func (r Retrier) retryNTimes(f RetryFunc) error {
	currentBackoff := r.exponentialBackoff

	for i := uint32(0); i < r.times; i++ {
		err := f()
		if err != nil {
			if r.isWhitelisted(err) {
				return err
			}

			log.Warn(err)
			currentBackoff = currentBackoff * time.Duration(r.exponentialFactor)
			time.Sleep(currentBackoff)
			continue
		}

		return nil
	}

	return ErrMaximumRetriesReached
}

func (r Retrier) retryUntilSuccess(f RetryFunc) error {
	currentBackoff := r.exponentialBackoff

	for {
		err := f()
		if err != nil {
			if r.isWhitelisted(err) {
				return err
			}

			log.Warn(err)
			currentBackoff = currentBackoff * time.Duration(r.exponentialFactor)
			time.Sleep(currentBackoff)
			continue
		}

		return nil
	}
}

func (r Retrier) isWhitelisted(err error) bool {
	_, ok := r.errWhitelist[err]
	return ok
}
