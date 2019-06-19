package netutil

import (
	"errors"
	"time"

	"github.com/prometheus/common/log"
)

var ErrMaximumRetriesReached = errors.New("maximum retries attempted without success")

type RetryFunc func() error

type Retrier struct {
	exponentialBackoff time.Duration
	exponentialFactor  uint32 // multiplier for the backoff duration that is applied on every retry. If 0 the backoff is not increased during retries
	times uint32 // number of times that the given function is going to be retried until success, if 0 it will be retried forever until success
	errWhitelist       map[error]struct{}
}

func NewRetrier(exponentialBackoff time.Duration, times, factor uint32) *Retrier {
	return &Retrier{
		exponentialBackoff: exponentialBackoff,
		times: 				times,
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
	if r.times == 0 {
		return r.retryUntilSuccess(f)
	}

	return r.retryNTimes(f)
}

func (r Retrier) retryNTimes(f RetryFunc) error {
	currentBackoff := r.exponentialBackoff

	for i:= uint32(0); i < r.times; i++ {
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
