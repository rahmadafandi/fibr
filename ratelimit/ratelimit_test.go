// Copyright 2026 Rahmad Afandi. MIT License.

package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLimiter(t *testing.T) (*Limiter, *time.Time) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	l := New(client)
	now := time.Unix(1000, 0)
	l.now = func() time.Time { return now }
	return l, &now
}

func TestAllowConsumesUntilEmpty(t *testing.T) {
	l, _ := newLimiter(t)
	ctx := context.Background()
	rule := Rule{Capacity: 3, RefillPerSec: 1}

	for i := range 3 {
		res, err := l.Allow(ctx, "k", rule, 1)
		require.NoError(t, err)
		assert.True(t, res.Allowed, "request %d should pass", i)
	}
	res, err := l.Allow(ctx, "k", rule, 1)
	require.NoError(t, err)
	assert.False(t, res.Allowed)
	assert.Equal(t, 0, res.Remaining)
	assert.Greater(t, res.RetryAfter, time.Duration(0))
}

func TestRefillOverTime(t *testing.T) {
	l, now := newLimiter(t)
	ctx := context.Background()
	rule := Rule{Capacity: 2, RefillPerSec: 1} // 1 token/sec

	_, _ = l.Allow(ctx, "k", rule, 1)
	_, _ = l.Allow(ctx, "k", rule, 1)
	res, _ := l.Allow(ctx, "k", rule, 1)
	require.False(t, res.Allowed)

	*now = now.Add(2 * time.Second) // refill 2 tokens
	res, err := l.Allow(ctx, "k", rule, 1)
	require.NoError(t, err)
	assert.True(t, res.Allowed)
}

func TestCostGreaterThanOne(t *testing.T) {
	l, _ := newLimiter(t)
	ctx := context.Background()
	rule := Rule{Capacity: 10, RefillPerSec: 1}

	res, err := l.Allow(ctx, "k", rule, 7)
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, 3, res.Remaining)

	// 4 > 3 remaining -> denied.
	res, err = l.Allow(ctx, "k", rule, 4)
	require.NoError(t, err)
	assert.False(t, res.Allowed)
}

func TestIndependentKeys(t *testing.T) {
	l, _ := newLimiter(t)
	ctx := context.Background()
	rule := Rule{Capacity: 1, RefillPerSec: 1}

	res, _ := l.Allow(ctx, "a", rule, 1)
	require.True(t, res.Allowed)
	res, _ = l.Allow(ctx, "b", rule, 1) // different bucket
	require.True(t, res.Allowed)
	res, _ = l.Allow(ctx, "a", rule, 1) // a exhausted
	require.False(t, res.Allowed)
}
