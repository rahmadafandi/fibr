// Copyright 2026 Rahmad Afandi. MIT License.

package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetGet(t *testing.T) {
	c := New[int]()
	c.Set("a", 1)
	v, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)

	_, ok = c.Get("missing")
	assert.False(t, ok)
}

func TestTTLExpiry(t *testing.T) {
	c := New[string](WithDefaultTTL(time.Minute))
	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }

	c.Set("k", "v")
	_, ok := c.Get("k")
	assert.True(t, ok)

	now = now.Add(2 * time.Minute) // past expiry
	_, ok = c.Get("k")
	assert.False(t, ok)
	assert.Equal(t, 0, c.Len(), "expired entry removed on Get")
}

func TestNoExpiryByDefault(t *testing.T) {
	c := New[int]()
	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }
	c.Set("k", 1)
	now = now.Add(1000 * time.Hour)
	_, ok := c.Get("k")
	assert.True(t, ok)
}

func TestLRUEviction(t *testing.T) {
	c := New[int](WithMaxSize(2))
	c.Set("a", 1)
	c.Set("b", 2)
	// Touch "a" so "b" becomes least-recently-used.
	_, _ = c.Get("a")
	c.Set("c", 3) // exceeds size 2 -> evict LRU ("b")

	_, ok := c.Get("b")
	assert.False(t, ok, "b should be evicted")
	_, ok = c.Get("a")
	assert.True(t, ok)
	_, ok = c.Get("c")
	assert.True(t, ok)
	assert.Equal(t, 2, c.Len())
}

func TestGetOrLoadCachesAndDedupes(t *testing.T) {
	c := New[string]()
	var calls int64
	loader := func() (string, error) {
		atomic.AddInt64(&calls, 1)
		time.Sleep(20 * time.Millisecond)
		return "value", nil
	}

	const n = 20
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := c.GetOrLoad(context.Background(), "k", loader)
			assert.NoError(t, err)
			assert.Equal(t, "value", v)
		}()
	}
	wg.Wait()
	assert.Equal(t, int64(1), atomic.LoadInt64(&calls))

	// Subsequent call is a cache hit (no new load).
	_, _ = c.GetOrLoad(context.Background(), "k", loader)
	assert.Equal(t, int64(1), atomic.LoadInt64(&calls))
}

func TestGetOrLoadErrorNotCached(t *testing.T) {
	c := New[int]()
	_, err := c.GetOrLoad(context.Background(), "k", func() (int, error) {
		return 0, assert.AnError
	})
	assert.Error(t, err)
	assert.Equal(t, 0, c.Len())
}

func TestDeleteAndLen(t *testing.T) {
	c := New[int]()
	c.Set("a", 1)
	c.Set("b", 2)
	assert.Equal(t, 2, c.Len())
	c.Delete("a")
	assert.Equal(t, 1, c.Len())
	_, ok := c.Get("a")
	assert.False(t, ok)
}

func TestJanitorSweepsExpired(t *testing.T) {
	// Use a real short TTL here (not an injected clock): the janitor goroutine
	// reads c.now concurrently, so reassigning it would race.
	c := New[int](WithJanitor(10 * time.Millisecond))
	t.Cleanup(c.Close)

	c.SetTTL("k", 1, 20*time.Millisecond)
	require.Eventually(t, func() bool { return c.Len() == 0 }, 2*time.Second, 5*time.Millisecond)
}
