// Copyright 2026 Rahmad Afandi. MIT License.

package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/rahmadafandi/fibr/v2/lock"
	"github.com/rahmadafandi/fibr/v2/logger"
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

	log         *logger.Logger
	maxAttempts int
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

// WithMaxAttempts moves an event to the outbox_dead table (see DeadEvent) once
// its publish has failed n times, so one poison event cannot block the queue
// forever. The count is stored in the outbox_attempts table, so it survives
// restarts and is shared by every replica; run Migrate to create that table.
// The default, 0, retries the head event indefinitely.
func WithMaxAttempts(n int) RelayOption {
	return func(r *Relay) { r.maxAttempts = n }
}

// WithLogger sets the logger Run reports Process errors to. The default is
// logger.Default().
func WithLogger(l *logger.Logger) RelayOption {
	return func(r *Relay) {
		if l != nil {
			r.log = l
		}
	}
}

// WithLock makes Process run each batch under a distributed lock, so only one
// replica relays at a time. The lock is held for ttl and renewed before every
// event, so ttl only needs to exceed the time to publish a single event.
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
		log:       logger.Default(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Process publishes one batch of pending events, oldest first, up to the
// configured batch size. Each event's PublishedAt is set only after its Publish
// succeeds; a Publish error stops the batch and is returned, leaving the failed
// event and the rest pending (unless WithMaxAttempts dead-letters it, in which
// case the batch continues). It returns the number of events published. When a
// Locker is configured and another replica holds the lock, Process returns
// (0, nil). Process must not be called concurrently on the same Relay.
func (r *Relay) Process(ctx context.Context) (int, error) {
	if r.locker == nil {
		return r.process(ctx, nil)
	}
	lk, ok, err := r.locker.TryAcquire(ctx, r.lockKey, r.lockTTL)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	defer func() { _ = lk.Release(context.WithoutCancel(ctx)) }()
	return r.process(ctx, func() error { return lk.Extend(ctx, r.lockTTL) })
}

// process runs one batch. renew, when non-nil, is called before each event to
// keep the relay lock alive; its failure means the lock was lost.
func (r *Relay) process(ctx context.Context, renew func() error) (int, error) {
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
		if renew != nil {
			if err := renew(); err != nil {
				return published, fmt.Errorf("outbox: renew relay lock: %w", err)
			}
		}
		if err := r.pub.Publish(ctx, e.Topic, e.Payload); err != nil {
			// A cancelled context (shutdown) says nothing about the event, so
			// it must not count towards WithMaxAttempts.
			if ctx.Err() != nil {
				return published, fmt.Errorf("outbox: publish event %d: %w", e.ID, err)
			}
			exhausted, aerr := r.exhausted(ctx, e.ID)
			if aerr != nil {
				return published, aerr
			}
			if !exhausted {
				return published, fmt.Errorf("outbox: publish event %d: %w", e.ID, err)
			}
			if derr := r.deadLetter(ctx, e, err); derr != nil {
				return published, derr
			}
			continue
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

// exhausted records one more failed attempt for event id and reports whether it
// has now used up WithMaxAttempts.
func (r *Relay) exhausted(ctx context.Context, id int64) (bool, error) {
	if r.maxAttempts <= 0 {
		return false, nil
	}
	a := &Attempt{EventID: id, Attempts: 1}
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(a).
			On("CONFLICT (event_id) DO UPDATE").
			Set("attempts = oa.attempts + 1").
			Exec(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(a).WherePK().Scan(ctx)
	})
	if err != nil {
		return false, fmt.Errorf("outbox: record attempt for event %d (has Migrate run?): %w", id, err)
	}
	return a.Attempts >= r.maxAttempts, nil
}

// deadLetter moves e from the outbox to outbox_dead, and drops its attempt
// counter, in one transaction.
func (r *Relay) deadLetter(ctx context.Context, e *Event, cause error) error {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		dead := &DeadEvent{
			ID:        e.ID,
			Topic:     e.Topic,
			Payload:   e.Payload,
			CreatedAt: e.CreatedAt,
			FailedAt:  time.Now(),
			Error:     cause.Error(),
		}
		if _, err := tx.NewInsert().Model(dead).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewDelete().Model(e).WherePK().Exec(ctx); err != nil {
			return err
		}
		_, err := tx.NewDelete().Model((*Attempt)(nil)).Where("event_id = ?", e.ID).Exec(ctx)
		return err
	})
	if err != nil {
		return fmt.Errorf("outbox: dead-letter event %d: %w", e.ID, err)
	}
	r.log.Error(cause, "outbox: event moved to dead-letter table", "event_id", e.ID, "topic", e.Topic)
	return nil
}

// Run calls Process every interval until ctx is cancelled, returning ctx.Err().
// A Process error is logged and the loop continues, so a transient publish
// failure does not stop the relay.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := r.Process(ctx); err != nil {
				r.log.Error(err, "outbox: relay process failed")
			}
		}
	}
}
