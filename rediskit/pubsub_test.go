// Copyright 2026 Rahmad Afandi. MIT License.

package rediskit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/v2/fibrtest"
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
