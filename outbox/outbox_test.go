// Copyright 2026 Rahmad Afandi. MIT License.

package outbox_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/rahmadafandi/fibr/outbox"
)

func newDB(t *testing.T) (*bun.DB, context.Context) {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	require.NoError(t, outbox.Migrate(ctx, db))
	return db, ctx
}

func pending(t *testing.T, db *bun.DB, ctx context.Context) []outbox.Event {
	t.Helper()
	var events []outbox.Event
	require.NoError(t, db.NewSelect().Model(&events).OrderExpr("id ASC").Scan(ctx))
	return events
}

func TestEnqueueInsertsPending(t *testing.T) {
	db, ctx := newDB(t)

	type orderCreated struct {
		OrderID int `json:"order_id"`
	}
	require.NoError(t, outbox.Enqueue(ctx, db, "order.created", orderCreated{OrderID: 42}))

	rows := pending(t, db, ctx)
	require.Len(t, rows, 1)
	assert.Equal(t, "order.created", rows[0].Topic)
	assert.Nil(t, rows[0].PublishedAt)
	assert.JSONEq(t, `{"order_id":42}`, string(rows[0].Payload))
	assert.False(t, rows[0].CreatedAt.IsZero())
}

func TestEnqueueRollbackLeavesNoRow(t *testing.T) {
	db, ctx := newDB(t)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, outbox.Enqueue(ctx, tx, "order.created", map[string]int{"x": 1}))
	require.NoError(t, tx.Rollback())

	assert.Empty(t, pending(t, db, ctx))
}

func TestEnqueueInTxCommits(t *testing.T) {
	db, ctx := newDB(t)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, outbox.Enqueue(ctx, tx, "order.created", map[string]int{"x": 1}))
	require.NoError(t, tx.Commit())

	assert.Len(t, pending(t, db, ctx), 1)
}

func TestMigrateIsIdempotent(t *testing.T) {
	db, ctx := newDB(t)
	// newDB already migrated once; a second call must not error.
	require.NoError(t, outbox.Migrate(ctx, db))
}
