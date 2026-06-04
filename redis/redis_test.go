// Copyright 2026 Rahmad Afandi. MIT License.

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) *Redis {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return New(client)
}

func TestSetGet(t *testing.T) {
	r := newTestRedis(t)
	ctx := context.Background()

	type item struct{ Name string }
	assert.NoError(t, r.Set(ctx, "k", item{Name: "x"}, time.Minute))

	var got item
	assert.NoError(t, r.Get(ctx, "k", &got))
	assert.Equal(t, "x", got.Name)
}

func TestRememberMissThenHit(t *testing.T) {
	r := newTestRedis(t)
	ctx := context.Background()

	calls := 0
	loader := func() (string, error) {
		calls++
		return "loaded", nil
	}

	v, err := Remember(ctx, r, "key", time.Minute, loader)
	assert.NoError(t, err)
	assert.Equal(t, "loaded", v)
	assert.Equal(t, 1, calls)

	v, err = Remember(ctx, r, "key", time.Minute, loader)
	assert.NoError(t, err)
	assert.Equal(t, "loaded", v)
	assert.Equal(t, 1, calls)
}

func TestRememberLoaderError(t *testing.T) {
	r := newTestRedis(t)
	ctx := context.Background()

	_, err := Remember(ctx, r, "key", time.Minute, func() (string, error) {
		return "", assert.AnError
	})
	assert.Error(t, err)
}

func TestParseRedisOptions(t *testing.T) {
	opt, err := ParseRedisOptions("redis://localhost:6379/0")
	assert.NoError(t, err)
	assert.NotNil(t, opt)

	bad, err := ParseRedisOptions("not-a-valid-url")
	assert.Error(t, err)
	assert.Nil(t, bad)
}
