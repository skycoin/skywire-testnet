package netutil

import (
	"errors"
	"github.com/prometheus/common/log"
	"time"
)

var ErrThresholdReached = errors.New("threshold timeout has been reached")

type RetryFunc func() error

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

func (r Retrier) Do(f RetryFunc) error {
	counter := time.Duration(0)
	currentBackoff := r.exponentialBackoff
	var err error

	t := time.NewTicker(r.exponentialBackoff)
	doneCh := time.After(r.threshold)
	errCh := make(chan error)
	go func() {
		errCh <- f()
	}()
	for {
		select {
		case <- doneCh:
			return ErrThresholdReached
		case <- t.C:
			counter += currentBackoff
			currentBackoff = currentBackoff * time.Duration(r.exponentialFactor)
			t.Stop()
			t = time.NewTicker(currentBackoff)

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
		}
	}
}


func (r Retrier) isWhitelisted(err error) bool {
	_, ok := r.errWhitelist[err]
	return ok
}