// Copyright 2026 Rahmad Afandi. MIT License.

package http

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned by request methods when the circuit breaker is open
// and the request is rejected without being sent.
var ErrCircuitOpen = errors.New("http: circuit breaker open")

type breakerState int

const (
	breakerClosed breakerState = iota
	breakerOpen
	breakerHalfOpen
)

// breaker is a per-client circuit breaker. It opens after maxFailures
// consecutive failures and rejects requests until openTimeout elapses, after
// which it allows a single probe (half-open).
type breaker struct {
	maxFailures int
	openTimeout time.Duration
	now         func() time.Time

	mu       sync.Mutex
	state    breakerState
	failures int
	openedAt time.Time
}

func newBreaker(maxFailures int, openTimeout time.Duration) *breaker {
	if maxFailures < 1 {
		maxFailures = 1
	}
	return &breaker{
		maxFailures: maxFailures,
		openTimeout: openTimeout,
		now:         time.Now,
	}
}

// allow reports whether a request may proceed. When the open timeout has
// elapsed it transitions to half-open and allows a single probe.
func (b *breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == breakerOpen {
		if b.now().Sub(b.openedAt) >= b.openTimeout {
			b.state = breakerHalfOpen
			return true
		}
		return false
	}
	// Closed allows; half-open rejects further calls until the probe resolves.
	return b.state == breakerClosed
}

// onSuccess records a successful outcome, closing the breaker.
func (b *breaker) onSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.state = breakerClosed
}

// onFailure records a failed outcome, opening the breaker when the failure
// threshold is reached (or immediately when a half-open probe fails).
func (b *breaker) onFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == breakerHalfOpen {
		b.state = breakerOpen
		b.openedAt = b.now()
		return
	}
	b.failures++
	if b.failures >= b.maxFailures {
		b.state = breakerOpen
		b.openedAt = b.now()
	}
}
