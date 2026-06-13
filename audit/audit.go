// Copyright 2026 Rahmad Afandi. MIT License.

// Package audit records an append-only actor/action/target trail through a
// pluggable Sink, with a Bun-backed sink and a Fiber helper that prefills
// request context.
package audit

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"

	fibrctx "github.com/rahmadafandi/fibr/context"
)

// Entry is one audit record.
type Entry struct {
	bun.BaseModel `bun:"table:audit_log,alias:al"`

	ID        int64          `bun:"id,pk,autoincrement"`
	Actor     string         `bun:"actor"`
	Action    string         `bun:"action,notnull"`
	Target    string         `bun:"target"`
	TargetID  string         `bun:"target_id"`
	Metadata  map[string]any `bun:"metadata"`
	IP        string         `bun:"ip"`
	RequestID string         `bun:"request_id"`
	CreatedAt time.Time      `bun:"created_at,notnull"`
}

// Sink persists audit entries.
type Sink interface {
	Record(ctx context.Context, e *Entry) error
}

// Recorder records entries through a Sink.
type Recorder struct {
	sink  Sink
	actor func(*fiber.Ctx) string
}

// Option configures a Recorder.
type Option func(*Recorder)

// WithActor sets how the actor is derived from a request (e.g. auth.Subject).
func WithActor(fn func(*fiber.Ctx) string) Option {
	return func(r *Recorder) { r.actor = fn }
}

// New returns a Recorder writing to sink.
func New(sink Sink, opts ...Option) *Recorder {
	r := &Recorder{sink: sink}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Record sets CreatedAt (when zero) and writes the entry to the sink.
func (r *Recorder) Record(ctx context.Context, e Entry) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	return r.sink.Record(ctx, &e)
}

// FromRequest returns an Entry prefilled with Actor (via the configured
// extractor, if any), IP, and RequestID. The caller fills Action/Target/etc and
// passes it to Record.
func (r *Recorder) FromRequest(c *fiber.Ctx) Entry {
	e := Entry{
		IP:        c.IP(),
		RequestID: fibrctx.GetRequestID(c),
	}
	if r.actor != nil {
		e.Actor = r.actor(c)
	}
	return e
}
