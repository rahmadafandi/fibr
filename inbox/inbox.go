// Copyright 2026 Rahmad Afandi. MIT License.

// Package inbox implements the idempotent-consumer pattern: a dedup table that
// lets a consumer process each message exactly once under at-least-once
// delivery. It is the consumer-side complement to the outbox package.
package inbox

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Entry is one processed-message marker.
type Entry struct {
	bun.BaseModel `bun:"table:inbox,alias:ib"`

	MessageID   string    `bun:"message_id,pk"`
	ProcessedAt time.Time `bun:"processed_at,notnull"`
}

// Migrate creates the inbox table if it does not exist.
func Migrate(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().Model((*Entry)(nil)).IfNotExists().Exec(ctx); err != nil {
		return fmt.Errorf("inbox: create table: %w", err)
	}
	return nil
}

// Once runs fn exactly once for messageID. It inserts a dedup marker and runs fn
// in a single transaction: a messageID already recorded skips fn and returns nil;
// if fn returns an error the transaction (marker included) rolls back, so the
// message can be reprocessed later. For that guarantee, fn must perform its work
// through the provided tx.
func Once(ctx context.Context, db *bun.DB, messageID string, fn func(ctx context.Context, tx bun.Tx) error) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewInsert().
			Model(&Entry{MessageID: messageID, ProcessedAt: time.Now()}).
			On("CONFLICT (message_id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("inbox: mark message: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("inbox: rows affected: %w", err)
		}
		if n == 0 {
			// Already processed: skip fn, commit the no-op transaction.
			return nil
		}
		return fn(ctx, tx)
	})
}

// Prune deletes markers processed more than olderThan ago and returns how many
// were removed. Markers are never deleted otherwise; keep them for at least as
// long as a message can still be redelivered.
func Prune(ctx context.Context, db bun.IDB, olderThan time.Duration) (int64, error) {
	res, err := db.NewDelete().
		Model((*Entry)(nil)).
		Where("processed_at < ?", time.Now().Add(-olderThan)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("inbox: prune: %w", err)
	}
	return res.RowsAffected()
}
