// Copyright 2026 Rahmad Afandi. MIT License.

package health

import (
	"context"
	"errors"
	"sync/atomic"
)

// ReadinessGate is a toggle that fails its readiness check while closed. Closing
// it during shutdown makes /readyz report not-ready before the server drains, so
// a load balancer stops routing new traffic first.
type ReadinessGate struct {
	closed atomic.Bool
}

// NewReadinessGate returns an open (ready) gate.
func NewReadinessGate() *ReadinessGate {
	return &ReadinessGate{}
}

// Check returns a NamedCheck that fails while the gate is closed.
func (g *ReadinessGate) Check() NamedCheck {
	return Check("readiness", func(_ context.Context) error {
		if g.closed.Load() {
			return errors.New("draining")
		}
		return nil
	})
}

// Close marks the gate draining, so the readiness check starts failing.
func (g *ReadinessGate) Close() { g.closed.Store(true) }

// Open marks the gate ready again.
func (g *ReadinessGate) Open() { g.closed.Store(false) }

// Ready reports whether the gate is open.
func (g *ReadinessGate) Ready() bool { return !g.closed.Load() }
