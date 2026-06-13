// Copyright 2026 Rahmad Afandi. MIT License.

package audit_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/rahmadafandi/fibr/audit"
)

func newDB(t *testing.T) (*bun.DB, context.Context) {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	require.NoError(t, audit.Migrate(ctx, db))
	return db, ctx
}

func TestRecordInsertsAndSetsCreatedAt(t *testing.T) {
	db, ctx := newDB(t)
	rec := audit.New(audit.NewBunSink(db))

	err := rec.Record(ctx, audit.Entry{
		Actor:    "u1",
		Action:   "order.delete",
		Target:   "order",
		TargetID: "42",
		Metadata: map[string]any{"reason": "fraud"},
	})
	require.NoError(t, err)

	got, err := audit.List(ctx, db, audit.Filter{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "u1", got[0].Actor)
	assert.Equal(t, "order.delete", got[0].Action)
	assert.Equal(t, "42", got[0].TargetID)
	assert.Equal(t, "fraud", got[0].Metadata["reason"])
	assert.False(t, got[0].CreatedAt.IsZero())
}

func TestMigrateIdempotent(t *testing.T) {
	db, ctx := newDB(t)
	require.NoError(t, audit.Migrate(ctx, db))
}

func TestListFiltersAndOrdersAndLimits(t *testing.T) {
	db, ctx := newDB(t)
	rec := audit.New(audit.NewBunSink(db))

	require.NoError(t, rec.Record(ctx, audit.Entry{Actor: "u1", Action: "login"}))
	require.NoError(t, rec.Record(ctx, audit.Entry{Actor: "u2", Action: "login"}))
	require.NoError(t, rec.Record(ctx, audit.Entry{Actor: "u1", Action: "logout"}))

	byActor, err := audit.List(ctx, db, audit.Filter{Actor: "u1"})
	require.NoError(t, err)
	assert.Len(t, byActor, 2)
	// Newest first.
	assert.Equal(t, "logout", byActor[0].Action)

	byAction, err := audit.List(ctx, db, audit.Filter{Action: "login"})
	require.NoError(t, err)
	assert.Len(t, byAction, 2)

	limited, err := audit.List(ctx, db, audit.Filter{Limit: 1})
	require.NoError(t, err)
	assert.Len(t, limited, 1)
}
