// Copyright 2026 Rahmad Afandi. MIT License.

// Package outbox implements the transactional-outbox pattern: events are
// written into an outbox table within the same database transaction as the
// business data, and a background Relay publishes pending events at-least-once.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Event is one row of the outbox. Payload holds the JSON-encoded event body.
// PublishedAt is nil while the event is pending and set once the relay has
// published it.
type Event struct {
	bun.BaseModel `bun:"table:outbox,alias:o"`

	ID          int64      `bun:"id,pk,autoincrement"`
	Topic       string     `bun:"topic,notnull"`
	Payload     []byte     `bun:"payload,notnull"`
	CreatedAt   time.Time  `bun:"created_at,notnull"`
	PublishedAt *time.Time `bun:"published_at"`
}

// DeadEvent is an event the relay gave up on (see WithMaxAttempts). It keeps
// the original ID, topic, payload, and creation time, plus when and why
// publishing was abandoned. Re-enqueue it with Enqueue to retry.
type DeadEvent struct {
	bun.BaseModel `bun:"table:outbox_dead,alias:od"`

	ID        int64     `bun:"id,pk"`
	Topic     string    `bun:"topic,notnull"`
	Payload   []byte    `bun:"payload,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull"`
	FailedAt  time.Time `bun:"failed_at,notnull"`
	Error     string    `bun:"error,notnull"`
}

// Migrate creates the outbox table (if it does not exist) with an index on
// (published_at, id) so pending-event scans stay cheap, and the outbox_dead
// table used by WithMaxAttempts.
func Migrate(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().Model((*DeadEvent)(nil)).IfNotExists().Exec(ctx); err != nil {
		return fmt.Errorf("outbox: create dead-letter table: %w", err)
	}
	if _, err := db.NewCreateTable().Model((*Event)(nil)).IfNotExists().Exec(ctx); err != nil {
		return fmt.Errorf("outbox: create table: %w", err)
	}
	if _, err := db.NewCreateIndex().
		Model((*Event)(nil)).
		Index("outbox_pending_idx").
		Column("published_at", "id").
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("outbox: create index: %w", err)
	}
	return nil
}

// Enqueue JSON-encodes payload and inserts a pending Event using db, which may
// be a *bun.DB or a bun.Tx. Pass your business transaction so the event commits
// atomically with your writes.
func Enqueue(ctx context.Context, db bun.IDB, topic string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("outbox: encode payload: %w", err)
	}
	e := &Event{
		Topic:     topic,
		Payload:   b,
		CreatedAt: time.Now(),
	}
	if _, err := db.NewInsert().Model(e).Exec(ctx); err != nil {
		return fmt.Errorf("outbox: insert event: %w", err)
	}
	return nil
}

// Prune deletes events published more than olderThan ago and returns how many
// were removed. Published rows are never deleted otherwise, so call it
// periodically (e.g. from a scheduled job) to keep the table small.
func Prune(ctx context.Context, db bun.IDB, olderThan time.Duration) (int64, error) {
	res, err := db.NewDelete().
		Model((*Event)(nil)).
		Where("published_at IS NOT NULL").
		Where("published_at < ?", time.Now().Add(-olderThan)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("outbox: prune: %w", err)
	}
	return res.RowsAffected()
}
