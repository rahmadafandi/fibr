// Copyright 2026 Rahmad Afandi. MIT License.

package inbox_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/rahmadafandi/fibr/inbox"
)

func newDB(t *testing.T) (*bun.DB, context.Context) {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	require.NoError(t, inbox.Migrate(ctx, db))
	return db, ctx
}

func TestOnceRunsThenSkipsDuplicate(t *testing.T) {
	db, ctx := newDB(t)

	runs := 0
	work := func(_ context.Context, _ bun.Tx) error { runs++; return nil }

	require.NoError(t, inbox.Once(ctx, db, "msg-1", work))
	require.NoError(t, inbox.Once(ctx, db, "msg-1", work)) // duplicate
	assert.Equal(t, 1, runs, "fn must run only once per message id")
}

func TestOnceDifferentIDsBothRun(t *testing.T) {
	db, ctx := newDB(t)
	runs := 0
	work := func(_ context.Context, _ bun.Tx) error { runs++; return nil }

	require.NoError(t, inbox.Once(ctx, db, "a", work))
	require.NoError(t, inbox.Once(ctx, db, "b", work))
	assert.Equal(t, 2, runs)
}

func TestOnceRollsBackOnError(t *testing.T) {
	db, ctx := newDB(t)

	boom := errors.New("handler failed")
	calls := 0
	work := func(_ context.Context, _ bun.Tx) error {
		calls++
		if calls == 1 {
			return boom // first attempt fails -> marker must roll back
		}
		return nil
	}

	err := inbox.Once(ctx, db, "msg-1", work)
	assert.ErrorIs(t, err, boom)

	// Retry: because the marker rolled back, fn runs again and now succeeds.
	require.NoError(t, inbox.Once(ctx, db, "msg-1", work))
	assert.Equal(t, 2, calls)

	// A third call is a no-op (already processed).
	require.NoError(t, inbox.Once(ctx, db, "msg-1", work))
	assert.Equal(t, 2, calls)
}

func TestMigrateIdempotent(t *testing.T) {
	db, ctx := newDB(t)
	require.NoError(t, inbox.Migrate(ctx, db))
}
