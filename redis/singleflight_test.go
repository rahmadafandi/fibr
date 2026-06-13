// Copyright 2026 Rahmad Afandi. MIT License.

package redis

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRememberSingleflightDedupesConcurrentMisses(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	r := New(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))

	var calls int64
	loader := func() (string, error) {
		atomic.AddInt64(&calls, 1)
		time.Sleep(30 * time.Millisecond) // widen the race window
		return "value", nil
	}

	const n = 25
	var wg sync.WaitGroup
	results := make([]string, n)
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			v, err := Remember(context.Background(), r, "key", time.Minute, loader)
			assert.NoError(t, err)
			results[i] = v
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), atomic.LoadInt64(&calls), "loader should run once")
	for _, v := range results {
		assert.Equal(t, "value", v)
	}
}
