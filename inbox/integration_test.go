// Copyright 2026 Rahmad Afandi. MIT License.

package inbox_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/rahmadafandi/fibr/v2/database"
	"github.com/rahmadafandi/fibr/v2/inbox"
)

// Runs only when FIBR_TEST_POSTGRES_DSN points at a disposable database.
func TestPostgresOnceIsExactlyOnceUnderConcurrency(t *testing.T) {
	dsn := os.Getenv("FIBR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("FIBR_TEST_POSTGRES_DSN not set")
	}
	db, err := database.NewBun(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	drop := func() { _, _ = db.NewDropTable().Model((*inbox.Entry)(nil)).IfExists().Exec(ctx) }
	drop()
	t.Cleanup(drop)
	require.NoError(t, inbox.Migrate(ctx, db))

	var runs atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := inbox.Once(ctx, db, "msg-1", func(context.Context, bun.Tx) error {
				runs.Add(1)
				time.Sleep(20 * time.Millisecond) // hold the transaction open
				return nil
			})
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, runs.Load())

	n, err := inbox.Prune(ctx, db, -time.Minute)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
}
