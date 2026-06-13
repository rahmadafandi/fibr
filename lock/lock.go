// Copyright 2026 Rahmad Afandi. MIT License.

package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultRetryDelay = 50 * time.Millisecond

// Locker acquires distributed locks backed by a single Redis instance.
type Locker struct {
	client     redis.UniversalClient
	retryDelay time.Duration
}

// Option configures a Locker.
type Option func(*Locker)

// WithRetryDelay sets how long Acquire waits between attempts while a lock is
// held. The default is 50ms.
func WithRetryDelay(d time.Duration) Option {
	return func(l *Locker) { l.retryDelay = d }
}

// New returns a Locker using the given Redis client.
func New(client redis.UniversalClient, opts ...Option) *Locker {
	l := &Locker{client: client, retryDelay: defaultRetryDelay}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// newToken returns a random hex owner token.
func newToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("lock: generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// TryAcquire makes one attempt to acquire key for ttl. It returns ok=false
// (and a nil error) when the lock is already held by another owner; a non-nil
// error indicates a Redis failure.
func (l *Locker) TryAcquire(ctx context.Context, key string, ttl time.Duration) (*Lock, bool, error) {
	token, err := newToken()
	if err != nil {
		return nil, false, err
	}
	ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return &Lock{client: l.client, key: key, token: token}, true, nil
}

// Acquire blocks until it acquires key for ttl, retrying every retryDelay,
// until the context is cancelled or its deadline passes. It returns
// ErrNotAcquired (wrapping ctx.Err()) if the context ends first.
func (l *Locker) Acquire(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	for {
		lk, ok, err := l.TryAcquire(ctx, key, ttl)
		if err != nil {
			// A context cancellation/deadline surfaces here as a Redis transport
			// error; report it as ErrNotAcquired so callers can match it.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, fmt.Errorf("%w: %w", ErrNotAcquired, ctxErr)
			}
			return nil, err
		}
		if ok {
			return lk, nil
		}
		timer := time.NewTimer(l.retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("%w: %w", ErrNotAcquired, ctx.Err())
		case <-timer.C:
		}
	}
}

// Do acquires key for ttl, runs fn, then releases the lock. It returns
// ErrNotAcquired without running fn if the lock is held by another owner.
// Release errors are ignored; the error from fn is returned.
func (l *Locker) Do(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	lk, ok, err := l.TryAcquire(ctx, key, ttl)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotAcquired
	}
	defer func() { _ = lk.Release(ctx) }()
	return fn()
}
