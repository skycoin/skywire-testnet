package netutil

import (
	"errors"
	"time"

	"github.com/prometheus/common/log"
)

var ErrThresholdReached = errors.New("threshold timeout has been reached")

type RetryFunc func() error

type Retrier struct {
	exponentialBackoff time.Duration
	exponentialFactor  uint32
	threshold          time.Duration
	errWhitelist       map[error]struct{}
}

func NewRetrier(exponentialBackoff, threshold time.Duration, factor uint32) *Retrier {
	return &Retrier{
		exponentialBackoff: exponentialBackoff,
		threshold:          threshold,
		exponentialFactor:  factor,
		errWhitelist:       make(map[error]struct{}),
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

func (r Retrier) Do(f RetryFunc) error {
	var err error
	var backoff <-chan time.Time
	var doneCh <-chan time.Time

	currentBackoff := r.exponentialBackoff

	errCh := make(chan error)
	go func() {
		errCh <- f()
	}()

	for {
		select {
		case <-doneCh:
			return ErrThresholdReached
		case <-backoff:
			go func() {
				errCh <- f()
			}()
		case err = <-errCh:
			if err != nil {
				if r.isWhitelisted(err) {
					return err
				}
			} else {
				return nil
			}
			log.Warn(err)

			backoff = time.After(currentBackoff)
			currentBackoff = currentBackoff * time.Duration(r.exponentialFactor)
			if doneCh == nil {
				doneCh = time.After(r.threshold)
			}
		}
	}
}

func (r Retrier) isWhitelisted(err error) bool {
	_, ok := r.errWhitelist[err]
	return ok
}
