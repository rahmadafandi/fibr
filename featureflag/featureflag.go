// Copyright 2026 Rahmad Afandi. MIT License.

// Package featureflag evaluates runtime feature flags (boolean, percentage
// rollout, and per-user/group targeting) through a pluggable provider, with
// built-in Static, Rules, and Redis providers and a Fiber helper.
package featureflag

import "context"

// Eval carries the attributes a provider targets on when deciding a flag.
type Eval struct {
	UserID string
	Groups []string
	Attrs  map[string]string
}

// Provider decides whether a flag is enabled for a given evaluation.
type Provider interface {
	Enabled(ctx context.Context, flag string, e Eval) bool
}

// Flags is the facade over a Provider.
type Flags struct {
	p Provider
}

// New returns a Flags backed by the given provider.
func New(p Provider) *Flags {
	return &Flags{p: p}
}

// Enabled reports whether flag is on for the evaluation e.
func (f *Flags) Enabled(ctx context.Context, flag string, e Eval) bool {
	if f == nil || f.p == nil {
		return false
	}
	return f.p.Enabled(ctx, flag, e)
}
