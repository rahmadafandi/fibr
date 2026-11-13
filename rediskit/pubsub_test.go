// Copyright 2026 Rahmad Afandi. MIT License.

package rediskit

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/v2/fibrtest"

	"github.com/rahmadafandi/fibr/v2/logger"
)

func TestPublishSubscribe(t *testing.T) {
	_, client := fibrtest.NewRedis(t)
	r := New(client)

	type event struct {
		Key string `json:"key"`
	}

	recv := make(chan event, 1)
	sub, err := Subscribe[event](context.Background(), r, "cache:invalidate", func(_ context.Context, e event) error {
		recv <- e
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Close() })

	require.NoError(t, r.Publish(context.Background(), "cache:invalidate", event{Key: "user:1"}))

	select {
	case got := <-recv:
		require.Equal(t, "user:1", got.Key)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pub/sub message")
	}
}

func TestSubscribeClose(t *testing.T) {
	_, client := fibrtest.NewRedis(t)
	r := New(client)

	sub, err := Subscribe[string](context.Background(), r, "ch", func(context.Context, string) error { return nil })
	require.NoError(t, err)
	require.NoError(t, sub.Close())
}

// syncBuf is a goroutine-safe log sink.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestSubscribeLogsDecodeErrorsToConfiguredLogger(t *testing.T) {
	_, client := fibrtest.NewRedis(t)
	var buf syncBuf
	r := New(client)
	r.Logger = logger.New(&buf, logger.ErrorLevel)

	sub, err := Subscribe[int](context.Background(), r, "numbers", func(context.Context, int) error { return nil })
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Close() })

	// Not a JSON number: the message is skipped and the decode error logged.
	require.NoError(t, client.Publish(context.Background(), "numbers", "not-a-number").Err())
	require.Eventually(t, func() bool { return strings.Contains(buf.String(), "decode pubsub message") },
		2*time.Second, 10*time.Millisecond)
}
