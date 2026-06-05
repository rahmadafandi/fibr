// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"context"
	"os"
	"testing"
	"time"

	fhredis "github.com/rahmadafandi/fiber-helpers/redis"
	redislib "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestMemoryStoreBlock(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	blocked, err := s.IsBlocked(ctx, "jti-1")
	require.NoError(t, err)
	require.False(t, blocked)

	require.NoError(t, s.Block(ctx, "jti-1", time.Minute))
	blocked, err = s.IsBlocked(ctx, "jti-1")
	require.NoError(t, err)
	require.True(t, blocked)
}

func TestMemoryStoreBlockExpiry(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	require.NoError(t, s.Block(ctx, "jti-2", -time.Second)) // already expired
	blocked, err := s.IsBlocked(ctx, "jti-2")
	require.NoError(t, err)
	require.False(t, blocked)
}

func TestMemoryStoreFamily(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	_, ok, err := s.Family(ctx, "fid-1")
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, s.SetFamily(ctx, "fid-1", "jti-a", time.Minute))
	jti, ok, err := s.Family(ctx, "fid-1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "jti-a", jti)

	require.NoError(t, s.SetFamily(ctx, "fid-1", "jti-b", time.Minute))
	jti, _, _ = s.Family(ctx, "fid-1")
	require.Equal(t, "jti-b", jti)

	require.NoError(t, s.RevokeFamily(ctx, "fid-1"))
	_, ok, _ = s.Family(ctx, "fid-1")
	require.False(t, ok)
}

func TestMemoryStoreFamilyExpiry(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	require.NoError(t, s.SetFamily(ctx, "fid-x", "jti", -time.Second))
	_, ok, err := s.Family(ctx, "fid-x")
	require.NoError(t, err)
	require.False(t, ok)
}

// Compile-time assertion that MemoryStore satisfies TokenStore.
var _ TokenStore = (*MemoryStore)(nil)

func TestRedisStore(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set; skipping RedisStore integration test")
	}
	opt, err := fhredis.ParseRedisOptions(url)
	require.NoError(t, err)
	r := fhredis.New(redislib.NewClient(opt))
	ctx := context.Background()
	require.NoError(t, r.Client.Ping(ctx).Err())

	s := NewRedisStore(r, WithStorePrefix("test:auth:"))
	t.Cleanup(func() {
		_ = r.Client.Del(context.Background(), s.blockKey("jti-r1"), s.familyKey("fid-r1")).Err()
	})

	require.NoError(t, s.Block(ctx, "jti-r1", time.Minute))
	blocked, err := s.IsBlocked(ctx, "jti-r1")
	require.NoError(t, err)
	require.True(t, blocked)

	notBlocked, err := s.IsBlocked(ctx, "jti-absent")
	require.NoError(t, err)
	require.False(t, notBlocked)

	require.NoError(t, s.SetFamily(ctx, "fid-r1", "jti-a", time.Minute))
	jti, ok, err := s.Family(ctx, "fid-r1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "jti-a", jti)

	require.NoError(t, s.RevokeFamily(ctx, "fid-r1"))
	_, ok, err = s.Family(ctx, "fid-r1")
	require.NoError(t, err)
	require.False(t, ok)
}

var _ TokenStore = (*RedisStore)(nil)
