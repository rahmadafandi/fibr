// Copyright 2026 Rahmad Afandi. MIT License.

package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	fhredis "github.com/rahmadafandi/fibr/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newR(t *testing.T) *fhredis.Redis {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	return fhredis.New(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))
}

func TestMSetMGet(t *testing.T) {
	r := newR(t)
	ctx := context.Background()
	require.NoError(t, r.MSet(ctx, map[string]any{"a": 1, "b": "two"}, 0))

	got, err := r.MGet(ctx, "a", "b", "missing")
	require.NoError(t, err)
	require.Equal(t, "1", got["a"])
	require.Equal(t, `"two"`, got["b"]) // JSON-encoded string
	require.NotContains(t, got, "missing")
}

func TestMGetEmpty(t *testing.T) {
	r := newR(t)
	got, err := r.MGet(context.Background())
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestIncrDecr(t *testing.T) {
	r := newR(t)
	ctx := context.Background()
	n, err := r.Incr(ctx, "c")
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	n, err = r.Incr(ctx, "c")
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	n, err = r.Decr(ctx, "c")
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
}

func TestSetNX(t *testing.T) {
	r := newR(t)
	ctx := context.Background()
	ok, err := r.SetNX(ctx, "k", "first", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = r.SetNX(ctx, "k", "second", time.Minute)
	require.NoError(t, err)
	require.False(t, ok) // already set
}

func TestGetSet(t *testing.T) {
	r := newR(t)
	ctx := context.Background()
	require.NoError(t, r.Set(ctx, "g", "old", 0))
	prev, err := r.GetSet(ctx, "g", "new")
	require.NoError(t, err)
	require.Equal(t, `"old"`, prev) // prior raw JSON value
}
