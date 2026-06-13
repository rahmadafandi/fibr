// Copyright 2026 Rahmad Afandi. MIT License.

package lock

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	c := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = c.Close() })
	return mr, c
}

func TestReleaseOwnerOnly(t *testing.T) {
	mr, c := newTestClient(t)
	ctx := context.Background()

	// Manually plant a lock owned by token "owner".
	require.NoError(t, c.Set(ctx, "k", "owner", time.Minute).Err())

	// A handle with a foreign token cannot release it.
	foreign := &Lock{client: c, key: "k", token: "intruder"}
	assert.ErrorIs(t, foreign.Release(ctx), ErrNotHeld)
	assert.True(t, mr.Exists("k"))

	// The real owner can release it.
	owner := &Lock{client: c, key: "k", token: "owner"}
	assert.NoError(t, owner.Release(ctx))
	assert.False(t, mr.Exists("k"))
}

func TestExtendRenewsAndFailsWhenLost(t *testing.T) {
	mr, c := newTestClient(t)
	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k", "owner", time.Minute).Err())

	owner := &Lock{client: c, key: "k", token: "owner"}
	require.NoError(t, owner.Extend(ctx, 2*time.Hour))
	assert.InDelta(t, (2 * time.Hour).Seconds(), mr.TTL("k").Seconds(), 5)

	// Once the key is gone, Extend reports ErrNotHeld.
	mr.Del("k")
	assert.ErrorIs(t, owner.Extend(ctx, time.Minute), ErrNotHeld)
}

func TestToken(t *testing.T) {
	lk := &Lock{token: "abc"}
	assert.Equal(t, "abc", lk.Token())
}

func TestTryAcquire(t *testing.T) {
	_, c := newTestClient(t)
	ctx := context.Background()
	l := New(c)

	lk, ok, err := l.TryAcquire(ctx, "job", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotEmpty(t, lk.Token())

	// Second attempt while held returns ok=false, no error.
	_, ok2, err := l.TryAcquire(ctx, "job", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok2)

	// After release, it can be acquired again.
	require.NoError(t, lk.Release(ctx))
	_, ok3, err := l.TryAcquire(ctx, "job", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok3)
}

func TestAcquireBlocksThenSucceeds(t *testing.T) {
	_, c := newTestClient(t)
	ctx := context.Background()
	l := New(c, WithRetryDelay(10*time.Millisecond))

	first, ok, err := l.TryAcquire(ctx, "job", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	released := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = first.Release(context.Background())
		close(released)
	}()

	lk, err := l.Acquire(ctx, "job", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lk)
	<-released
}

func TestAcquireRespectsContextDeadline(t *testing.T) {
	_, c := newTestClient(t)
	l := New(c, WithRetryDelay(10*time.Millisecond))

	held, ok, err := l.TryAcquire(context.Background(), "job", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	defer func() { _ = held.Release(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = l.Acquire(ctx, "job", time.Minute)
	assert.ErrorIs(t, err, ErrNotAcquired)
}

func TestDoRunsOnceAndSkipsWhenHeld(t *testing.T) {
	_, c := newTestClient(t)
	ctx := context.Background()
	l := New(c)

	ran := false
	require.NoError(t, l.Do(ctx, "job", time.Minute, func() error {
		ran = true
		return nil
	}))
	assert.True(t, ran)

	// Plant a competing holder so Do skips.
	require.NoError(t, c.Set(ctx, "job", "other", time.Minute).Err())
	ranAgain := false
	err := l.Do(ctx, "job", time.Minute, func() error {
		ranAgain = true
		return nil
	})
	assert.ErrorIs(t, err, ErrNotAcquired)
	assert.False(t, ranAgain)
}

func TestConcurrentSingleWinner(t *testing.T) {
	_, c := newTestClient(t)
	l := New(c)

	const n = 20
	var wins int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, ok, _ := l.TryAcquire(context.Background(), "race", time.Minute); ok {
				atomic.AddInt64(&wins, 1)
			}
		}()
	}
	close(start)
	wg.Wait()
	assert.Equal(t, int64(1), wins)
}
