// Copyright 2026 Rahmad Afandi. MIT License.

package outbox_test

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/v2/database"
	"github.com/rahmadafandi/fibr/v2/logger"
	"github.com/rahmadafandi/fibr/v2/outbox"
)

// Runs only when FIBR_TEST_POSTGRES_DSN points at a disposable database.
func TestPostgresRelayLifecycle(t *testing.T) {
	dsn := os.Getenv("FIBR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("FIBR_TEST_POSTGRES_DSN not set")
	}
	db, err := database.NewBun(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	drop := func() {
		_, _ = db.NewDropTable().Model((*outbox.Event)(nil)).IfExists().Exec(ctx)
		_, _ = db.NewDropTable().Model((*outbox.DeadEvent)(nil)).IfExists().Exec(ctx)
		_, _ = db.NewDropTable().Model((*outbox.Attempt)(nil)).IfExists().Exec(ctx)
	}
	drop()
	t.Cleanup(drop)
	require.NoError(t, outbox.Migrate(ctx, db))
	require.NoError(t, outbox.Migrate(ctx, db), "Migrate must be idempotent")

	for _, topic := range []string{"poison", "ok"} {
		require.NoError(t, outbox.Enqueue(ctx, db, topic, map[string]string{"t": topic}))
	}
	pub := funcPublisher(func(topic string) error {
		if topic == "poison" {
			return errors.New("publish boom")
		}
		return nil
	})
	relay := outbox.NewRelay(db, pub, outbox.WithMaxAttempts(3),
		outbox.WithLogger(logger.New(io.Discard, logger.ErrorLevel)))

	// Two failures are counted in the database (insert, then upsert)...
	for range 2 {
		_, err := relay.Process(ctx)
		require.Error(t, err)
	}
	var attempts []outbox.Attempt
	require.NoError(t, db.NewSelect().Model(&attempts).Scan(ctx))
	require.Len(t, attempts, 1)
	assert.Equal(t, 2, attempts[0].Attempts)

	// ...and the third dead-letters the event and lets the queue move on.
	n, err := relay.Process(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	dead, err := db.NewSelect().Model((*outbox.DeadEvent)(nil)).Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, dead)

	pruned, err := outbox.Prune(ctx, db, -time.Minute) // cutoff in the future: everything published
	require.NoError(t, err)
	assert.EqualValues(t, 1, pruned)
	assert.Empty(t, pending(t, db, ctx))
}
