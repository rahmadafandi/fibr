// Copyright 2026 Rahmad Afandi. MIT License.

package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rahmadafandi/fibr/lock"
	"github.com/rahmadafandi/fibr/logger"
	"github.com/uptrace/bun"
)

const (
	defaultBatchSize = 100
	defaultInterval  = time.Second
)

// Publisher publishes a single event's raw payload bytes to its topic.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

// Relay reads pending events from the outbox and publishes them.
type Relay struct {
	db        *bun.DB
	pub       Publisher
	batchSize int
	interval  time.Duration

	locker  *lock.Locker
	lockKey string
	lockTTL time.Duration
}

// RelayOption configures a Relay.
type RelayOption func(*Relay)

// WithBatchSize sets the maximum number of events published per Process call.
// The default is 100.
func WithBatchSize(n int) RelayOption {
	return func(r *Relay) {
		if n > 0 {
			r.batchSize = n
		}
	}
}

// WithInterval sets how often Run calls Process. The default is one second.
func WithInterval(d time.Duration) RelayOption {
	return func(r *Relay) {
		if d > 0 {
			r.interval = d
		}
	}
}

// WithLock makes Process run each batch under a distributed lock, so only one
// replica relays at a time. key and ttl are passed to lock.Do.
func WithLock(l *lock.Locker, key string, ttl time.Duration) RelayOption {
	return func(r *Relay) {
		r.locker = l
		r.lockKey = key
		r.lockTTL = ttl
	}
}

// NewRelay returns a Relay that publishes outbox events from db via pub.
func NewRelay(db *bun.DB, pub Publisher, opts ...RelayOption) *Relay {
	r := &Relay{
		db:        db,
		pub:       pub,
		batchSize: defaultBatchSize,
		interval:  defaultInterval,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Process publishes one batch of pending events, oldest first, up to the
// configured batch size. Each event's PublishedAt is set only after its Publish
// succeeds; a Publish error stops the batch and is returned, leaving the failed
// event and the rest pending. It returns the number of events published. When a
// Locker is configured and another replica holds the lock, Process returns
// (0, nil).
func (r *Relay) Process(ctx context.Context) (int, error) {
	if r.locker == nil {
		return r.process(ctx)
	}
	var n int
	err := r.locker.Do(ctx, r.lockKey, r.lockTTL, func() error {
		var perr error
		n, perr = r.process(ctx)
		return perr
	})
	if errors.Is(err, lock.ErrNotAcquired) {
		return 0, nil
	}
	return n, err
}

func (r *Relay) process(ctx context.Context) (int, error) {
	var events []Event
	if err := r.db.NewSelect().
		Model(&events).
		Where("published_at IS NULL").
		OrderExpr("id ASC").
		Limit(r.batchSize).
		Scan(ctx); err != nil {
		return 0, fmt.Errorf("outbox: load pending: %w", err)
	}

	published := 0
	for i := range events {
		e := &events[i]
		if err := r.pub.Publish(ctx, e.Topic, e.Payload); err != nil {
			return published, fmt.Errorf("outbox: publish event %d: %w", e.ID, err)
		}
		now := time.Now()
		if _, err := r.db.NewUpdate().
			Model(e).
			Set("published_at = ?", now).
			WherePK().
			Exec(ctx); err != nil {
			return published, fmt.Errorf("outbox: mark event %d published: %w", e.ID, err)
		}
		published++
	}
	return published, nil
}

// Run calls Process every interval until ctx is cancelled, returning ctx.Err().
// A Process error is logged and the loop continues, so a transient publish
// failure does not stop the relay.
func (r *Relay) Run(ctx context.Context) error {
	log := logger.Default()
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := r.Process(ctx); err != nil {
				log.Error(err, "outbox: relay process failed")
			}
		}
	}
}
