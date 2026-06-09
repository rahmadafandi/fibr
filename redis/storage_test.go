// Copyright 2026 Rahmad Afandi. MIT License.

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Storage must satisfy the fiber.Storage interface.
var _ fiber.Storage = (*Storage)(nil)

func newTestStorage(t *testing.T) (*Storage, *miniredis.Miniredis, *Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return NewStorage(client), mr, client
}

func TestStorageSetGet(t *testing.T) {
	s, mr, _ := newTestStorage(t)
	require.NoError(t, s.Set("k", []byte("v"), time.Minute))
	got, err := s.Get("k")
	require.NoError(t, err)
	assert.Equal(t, []byte("v"), got)
	assert.True(t, mr.Exists("fiber:storage:k"))
}

func TestStorageGetMiss(t *testing.T) {
	s, _, _ := newTestStorage(t)
	got, err := s.Get("nope")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStorageDelete(t *testing.T) {
	s, _, _ := newTestStorage(t)
	require.NoError(t, s.Set("k", []byte("v"), 0))
	require.NoError(t, s.Delete("k"))
	got, _ := s.Get("k")
	assert.Nil(t, got)
}

func TestStorageResetScoped(t *testing.T) {
	s, _, client := newTestStorage(t)
	require.NoError(t, s.Set("a", []byte("1"), 0))
	require.NoError(t, s.Set("b", []byte("2"), 0))
	require.NoError(t, client.Set(context.Background(), "other", "keep", 0).Err())

	require.NoError(t, s.Reset())

	ga, _ := s.Get("a")
	assert.Nil(t, ga)
	gb, _ := s.Get("b")
	assert.Nil(t, gb)
	val, err := client.Get(context.Background(), "other").Result()
	require.NoError(t, err)
	assert.Equal(t, "keep", val)
}

func TestStorageSetExpiry(t *testing.T) {
	s, mr, _ := newTestStorage(t)
	require.NoError(t, s.Set("k", []byte("v"), time.Minute))
	mr.FastForward(2 * time.Minute)
	got, err := s.Get("k")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStorageWithPrefix(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	s := NewStorage(client, WithPrefix("rl:"))
	require.NoError(t, s.Set("k", []byte("v"), 0))
	assert.True(t, mr.Exists("rl:k"))
}

func TestStorageEmptyKeyNoop(t *testing.T) {
	s, _, _ := newTestStorage(t)
	got, err := s.Get("")
	require.NoError(t, err)
	assert.Nil(t, got)
	require.NoError(t, s.Set("", []byte("v"), 0))
	require.NoError(t, s.Delete(""))
}
