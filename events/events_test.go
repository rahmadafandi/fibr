// Copyright 2026 Rahmad Afandi. MIT License.

package events_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/events"
	"github.com/rahmadafandi/fibr/logger"
)

type orderCreated struct{ ID int }
type userRegistered struct{ Email string }

func TestSyncPublishRunsAllHandlersInOrder(t *testing.T) {
	b := events.New()
	var order []string
	events.Subscribe(b, func(_ context.Context, e orderCreated) error {
		order = append(order, "a")
		return nil
	})
	events.Subscribe(b, func(_ context.Context, e orderCreated) error {
		order = append(order, "b")
		return nil
	})

	err := events.Publish(context.Background(), b, orderCreated{ID: 1})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, order)
}

func TestSyncPublishJoinsErrors(t *testing.T) {
	b := events.New()
	errA := errors.New("a failed")
	errB := errors.New("b failed")
	events.Subscribe(b, func(_ context.Context, e orderCreated) error { return errA })
	events.Subscribe(b, func(_ context.Context, e orderCreated) error { return errB })

	err := events.Publish(context.Background(), b, orderCreated{})
	require.Error(t, err)
	assert.ErrorIs(t, err, errA)
	assert.ErrorIs(t, err, errB)
}

func TestPublishNoHandlers(t *testing.T) {
	b := events.New()
	assert.NoError(t, events.Publish(context.Background(), b, orderCreated{}))
}

func TestTypeIsolation(t *testing.T) {
	b := events.New()
	called := false
	events.Subscribe(b, func(_ context.Context, e userRegistered) error {
		called = true
		return nil
	})

	require.NoError(t, events.Publish(context.Background(), b, orderCreated{ID: 1}))
	assert.False(t, called, "userRegistered handler must not see orderCreated")
}

func TestAsyncPublishReturnsImmediatelyAndRuns(t *testing.T) {
	b := events.New(events.WithAsync())
	done := make(chan int, 1)
	events.Subscribe(b, func(_ context.Context, e orderCreated) error {
		done <- e.ID
		return nil
	})

	require.NoError(t, events.Publish(context.Background(), b, orderCreated{ID: 42}))
	select {
	case id := <-done:
		assert.Equal(t, 42, id)
	case <-time.After(2 * time.Second):
		t.Fatal("async handler did not run")
	}
}

// safeBuffer is a mutex-guarded buffer so the async logging goroutine and the
// test can touch it without racing.
type safeBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestAsyncHandlerErrorIsLogged(t *testing.T) {
	buf := &safeBuffer{}
	log := logger.New(buf, zerolog.ErrorLevel)
	b := events.New(events.WithAsync(), events.WithLogger(log))

	events.Subscribe(b, func(_ context.Context, e orderCreated) error {
		return errors.New("boom")
	})

	require.NoError(t, events.Publish(context.Background(), b, orderCreated{}))
	assert.Eventually(t, func() bool {
		return strings.Contains(buf.String(), "async handler failed")
	}, 2*time.Second, 10*time.Millisecond)
}

func TestConcurrentPublishIsRaceSafe(t *testing.T) {
	b := events.New()
	var mu sync.Mutex
	count := 0
	events.Subscribe(b, func(_ context.Context, e orderCreated) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	})

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = events.Publish(context.Background(), b, orderCreated{})
		}()
	}
	wg.Wait()
	assert.Equal(t, 50, count)
}
