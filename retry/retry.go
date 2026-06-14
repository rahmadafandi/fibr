// Copyright 2026 Rahmad Afandi. MIT License.

// Package retry runs a function with configurable retries, exponential backoff,
// and optional jitter.
package retry

import (
	"context"
	"math/rand/v2"
	"time"
)

type config struct {
	attempts   int
	delay      time.Duration
	maxDelay   time.Duration
	multiplier float64
	jitter     float64
	retryIf    func(error) bool
}

// Option configures a retry.
type Option func(*config)

// WithAttempts sets the total number of attempts including the first. Values
// below 1 are clamped to 1. The default is 3.
func WithAttempts(n int) Option { return func(c *config) { c.attempts = n } }

// WithDelay sets the base backoff delay. The default is 100ms.
func WithDelay(d time.Duration) Option { return func(c *config) { c.delay = d } }

// WithMaxDelay caps the per-attempt backoff delay. Zero (default) means no cap.
func WithMaxDelay(d time.Duration) Option { return func(c *config) { c.maxDelay = d } }

// WithMultiplier sets the exponential backoff factor. The default is 2.0.
func WithMultiplier(m float64) Option { return func(c *config) { c.multiplier = m } }

// WithJitter adds random jitter as a fraction (0..1) of each delay. The default
// is 0 (no jitter).
func WithJitter(frac float64) Option { return func(c *config) { c.jitter = frac } }

// WithRetryIf retries only when pred(err) is true. The default retries on any
// non-nil error.
func WithRetryIf(pred func(error) bool) Option { return func(c *config) { c.retryIf = pred } }

func newConfig(opts []Option) config {
	c := config{
		attempts:   3,
		delay:      100 * time.Millisecond,
		multiplier: 2.0,
		retryIf:    func(error) bool { return true },
	}
	for _, opt := range opts {
		opt(&c)
	}
	if c.attempts < 1 {
		c.attempts = 1
	}
	return c
}

// Do runs fn, retrying on error per the options. It returns nil on the first
// success, the last error once attempts are exhausted (or pred is false), or
// ctx.Err() if the context ends.
func Do(ctx context.Context, fn func() error, opts ...Option) error {
	c := newConfig(opts)

	var lastErr error
	for i := 0; i < c.attempts; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if i == c.attempts-1 || !c.retryIf(lastErr) {
			return lastErr
		}
		if err := sleep(ctx, c.backoff(i)); err != nil {
			return err
		}
	}
	return lastErr
}

// DoValue is Do for a function that returns a value.
func DoValue[T any](ctx context.Context, fn func() (T, error), opts ...Option) (T, error) {
	var out T
	err := Do(ctx, func() error {
		v, err := fn()
		if err != nil {
			return err
		}
		out = v
		return nil
	}, opts...)
	if err != nil {
		var zero T
		return zero, err
	}
	return out, nil
}

// backoff returns the delay before the retry following attempt i (0-based).
func (c config) backoff(i int) time.Duration {
	d := float64(c.delay)
	for range i {
		d *= c.multiplier
	}
	if c.maxDelay > 0 && d > float64(c.maxDelay) {
		d = float64(c.maxDelay)
	}
	if c.jitter > 0 {
		d *= 1 - c.jitter + 2*c.jitter*rand.Float64()
	}
	return time.Duration(d)
}

// sleep waits for d or until ctx is done.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
