// Copyright 2026 Rahmad Afandi. MIT License.

package ratelimit

import (
	"context"
	"os"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Runs only when FIBR_TEST_REDIS_URL points at a Redis the test may write to.
// It exercises the Lua script on a real server rather than miniredis.
func TestRealRedisTokenBucket(t *testing.T) {
	url := os.Getenv("FIBR_TEST_REDIS_URL")
	if url == "" {
		t.Skip("FIBR_TEST_REDIS_URL not set")
	}
	opt, err := goredis.ParseURL(url)
	require.NoError(t, err)
	client := goredis.NewClient(opt)
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	const key = "fibr-integration"
	del := func() { _ = client.Del(ctx, keyPrefix+key).Err() }
	del()
	t.Cleanup(del)

	l := New(client)
	rule := Rule{Capacity: 3, RefillPerSec: 0.001}
	for i := range 3 {
		res, err := l.Allow(ctx, key, rule, 1)
		require.NoError(t, err)
		assert.True(t, res.Allowed)
		assert.Equal(t, 2-i, res.Remaining)
	}
	res, err := l.Allow(ctx, key, rule, 1)
	require.NoError(t, err)
	assert.False(t, res.Allowed)
	assert.Positive(t, res.RetryAfter)
}
