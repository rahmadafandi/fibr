// Copyright 2026 Rahmad Afandi. MIT License.

package outbox_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/lock"
	"github.com/rahmadafandi/fibr/outbox"
)

type call struct {
	Topic   string
	Payload []byte
}

type fakePublisher struct {
	mu     sync.Mutex
	calls  []call
	failAt int // 1-based call index that returns an error; 0 = never fail
}

func (f *fakePublisher) Publish(_ context.Context, topic string, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call{Topic: topic, Payload: append([]byte(nil), payload...)})
	if f.failAt != 0 && len(f.calls) == f.failAt {
		return errors.New("publish boom")
	}
	return nil
}

func (f *fakePublisher) topics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	for i, c := range f.calls {
		out[i] = c.Topic
	}
	return out
}

func TestProcessPublishesAllAndMarks(t *testing.T) {
	db, ctx := newDB(t)
	for _, topic := range []string{"a", "b", "c"} {
		require.NoError(t, outbox.Enqueue(ctx, db, topic, map[string]string{"t": topic}))
	}

	pub := &fakePublisher{}
	relay := outbox.NewRelay(db, pub)

	n, err := relay.Process(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []string{"a", "b", "c"}, pub.topics())

	for _, e := range pending(t, db, ctx) {
		assert.NotNil(t, e.PublishedAt, "event %d should be marked published", e.ID)
	}

	// Nothing left to publish.
	n, err = relay.Process(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestProcessOldestFirstAndBatchCap(t *testing.T) {
	db, ctx := newDB(t)
	for _, topic := range []string{"e1", "e2", "e3", "e4", "e5"} {
		require.NoError(t, outbox.Enqueue(ctx, db, topic, nil))
	}

	pub := &fakePublisher{}
	relay := outbox.NewRelay(db, pub, outbox.WithBatchSize(2))

	n, err := relay.Process(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []string{"e1", "e2"}, pub.topics())

	var unpublished int
	for _, e := range pending(t, db, ctx) {
		if e.PublishedAt == nil {
			unpublished++
		}
	}
	assert.Equal(t, 3, unpublished)
}

func TestProcessStopsOnPublishError(t *testing.T) {
	db, ctx := newDB(t)
	for _, topic := range []string{"a", "b", "c"} {
		require.NoError(t, outbox.Enqueue(ctx, db, topic, nil))
	}

	pub := &fakePublisher{failAt: 2}
	relay := outbox.NewRelay(db, pub)

	n, err := relay.Process(ctx)
	require.Error(t, err)
	assert.Equal(t, 1, n) // only the first event committed

	var published, unpublished int
	for _, e := range pending(t, db, ctx) {
		if e.PublishedAt == nil {
			unpublished++
		} else {
			published++
		}
	}
	assert.Equal(t, 1, published)
	assert.Equal(t, 2, unpublished)
}

func TestProcessWithLockSkipsWhenHeld(t *testing.T) {
	db, ctx := newDB(t)
	require.NoError(t, outbox.Enqueue(ctx, db, "a", nil))

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	const key = "outbox:relay"
	// A different owner already holds the lock.
	require.NoError(t, client.Set(ctx, key, "other-owner", time.Minute).Err())

	pub := &fakePublisher{}
	relay := outbox.NewRelay(db, pub, outbox.WithLock(lock.New(client), key, 30*time.Second))

	n, err := relay.Process(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, pub.topics())

	// Event remains pending.
	rows := pending(t, db, ctx)
	require.Len(t, rows, 1)
	assert.Nil(t, rows[0].PublishedAt)
}

func TestRedisPublisherSendsRawBytes(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	sub := client.Subscribe(ctx, "order.created")
	t.Cleanup(func() { _ = sub.Close() })
	_, err = sub.Receive(ctx) // wait for subscribe confirmation
	require.NoError(t, err)

	pub := outbox.NewRedisPublisher(client)
	payload := []byte(`{"order_id":42}`)
	require.NoError(t, pub.Publish(ctx, "order.created", payload))

	select {
	case msg := <-sub.Channel():
		assert.Equal(t, string(payload), msg.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published message")
	}
}
