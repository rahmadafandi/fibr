// Copyright 2026 Rahmad Afandi. MIT License.

package lock

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Runs only when FIBR_TEST_REDIS_URL points at a Redis the test may write to.
// It exercises the release/extend scripts on a real server rather than miniredis.
func TestRealRedisLockLifecycle(t *testing.T) {
	url := os.Getenv("FIBR_TEST_REDIS_URL")
	if url == "" {
		t.Skip("FIBR_TEST_REDIS_URL not set")
	}
	opt, err := redis.ParseURL(url)
	require.NoError(t, err)
	client := redis.NewClient(opt)
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	const key = "fibr-integration:lock"
	t.Cleanup(func() { _ = client.Del(ctx, key).Err() })
	_ = client.Del(ctx, key).Err()

	l := New(client)
	lk, ok, err := l.TryAcquire(ctx, key, time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	_, ok, err = l.TryAcquire(ctx, key, time.Second)
	require.NoError(t, err)
	assert.False(t, ok, "a held lock must not be acquired twice")

	require.NoError(t, lk.Extend(ctx, time.Minute))
	ttl, err := client.PTTL(ctx, key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 30*time.Second)

	require.NoError(t, lk.Release(ctx))
	assert.ErrorIs(t, lk.Release(ctx), ErrNotHeld)
	assert.ErrorIs(t, lk.Extend(ctx, time.Minute), ErrNotHeld)
}
