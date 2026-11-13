// Copyright 2026 Rahmad Afandi. MIT License.

package outbox_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/v2/lock"
	"github.com/rahmadafandi/fibr/v2/logger"
	"github.com/rahmadafandi/fibr/v2/outbox"

	"github.com/rahmadafandi/fibr/v2/fibrtest"
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

	_, client := fibrtest.NewRedis(t)

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
	_, client := fibrtest.NewRedis(t)

	ctx := context.Background()
	sub := client.Subscribe(ctx, "order.created")
	t.Cleanup(func() { _ = sub.Close() })
	_, err := sub.Receive(ctx) // wait for subscribe confirmation
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

// funcPublisher adapts a function to outbox.Publisher.
type funcPublisher func(topic string) error

func (f funcPublisher) Publish(_ context.Context, topic string, _ []byte) error { return f(topic) }

func TestMaxAttemptsDeadLettersPoisonEvent(t *testing.T) {
	db, ctx := newDB(t)
	for _, topic := range []string{"poison", "ok"} {
		require.NoError(t, outbox.Enqueue(ctx, db, topic, map[string]string{"t": topic}))
	}

	var delivered []string
	pub := funcPublisher(func(topic string) error {
		if topic == "poison" {
			return errors.New("publish boom")
		}
		delivered = append(delivered, topic)
		return nil
	})
	relay := outbox.NewRelay(db, pub, outbox.WithMaxAttempts(3),
		outbox.WithLogger(logger.New(io.Discard, logger.ErrorLevel)))

	// The first two failures keep the event at the head and block the queue.
	for range 2 {
		n, err := relay.Process(ctx)
		require.Error(t, err)
		assert.Equal(t, 0, n)
	}
	assert.Empty(t, delivered)

	// The third failure dead-letters it and the batch carries on.
	n, err := relay.Process(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"ok"}, delivered)

	var dead []outbox.DeadEvent
	require.NoError(t, db.NewSelect().Model(&dead).Scan(ctx))
	require.Len(t, dead, 1)
	assert.Equal(t, "poison", dead[0].Topic)
	assert.JSONEq(t, `{"t":"poison"}`, string(dead[0].Payload))
	assert.Contains(t, dead[0].Error, "publish boom")

	rows := pending(t, db, ctx)
	require.Len(t, rows, 1, "the poison event must be gone from the outbox")
	assert.Equal(t, "ok", rows[0].Topic)
}

func TestPruneDeletesOnlyOldPublished(t *testing.T) {
	db, ctx := newDB(t)
	for _, topic := range []string{"old", "recent", "pending"} {
		require.NoError(t, outbox.Enqueue(ctx, db, topic, nil))
	}
	mark := func(topic string, at time.Time) {
		_, err := db.NewUpdate().Model((*outbox.Event)(nil)).
			Set("published_at = ?", at).Where("topic = ?", topic).Exec(ctx)
		require.NoError(t, err)
	}
	mark("old", time.Now().Add(-48*time.Hour))
	mark("recent", time.Now().Add(-time.Minute))

	n, err := outbox.Prune(ctx, db, 24*time.Hour)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	var topics []string
	for _, e := range pending(t, db, ctx) {
		topics = append(topics, e.Topic)
	}
	assert.Equal(t, []string{"recent", "pending"}, topics)
}

func TestProcessWithLockReleasesAndStopsWhenLockLost(t *testing.T) {
	db, ctx := newDB(t)
	for _, topic := range []string{"a", "b"} {
		require.NoError(t, outbox.Enqueue(ctx, db, topic, nil))
	}

	mr, client := fibrtest.NewRedis(t)
	const key = "outbox:relay"

	// Losing the lock mid-batch (here: another owner takes it during the first
	// publish) must stop the batch before the next event.
	var delivered []string
	steal := funcPublisher(func(topic string) error {
		delivered = append(delivered, topic)
		mr.Set(key, "other-owner")
		return nil
	})
	relay := outbox.NewRelay(db, steal, outbox.WithLock(lock.New(client), key, 30*time.Second))
	n, err := relay.Process(ctx)
	require.ErrorIs(t, err, lock.ErrNotHeld)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"a"}, delivered)

	// With the lock free again, the rest is published and the lock released.
	mr.Del(key)
	relay = outbox.NewRelay(db, funcPublisher(func(string) error { return nil }),
		outbox.WithLock(lock.New(client), key, 30*time.Second))
	n, err = relay.Process(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.False(t, mr.Exists(key), "relay lock must be released after the batch")
}

func TestRunLogsProcessErrorsAndStopsOnCancel(t *testing.T) {
	db, _ := newDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, outbox.Enqueue(ctx, db, "a", nil))

	var buf syncBuffer
	pub := funcPublisher(func(string) error { cancel(); return errors.New("publish boom") })
	relay := outbox.NewRelay(db, pub, outbox.WithInterval(time.Millisecond),
		outbox.WithLogger(logger.New(&buf, logger.ErrorLevel)))

	assert.ErrorIs(t, relay.Run(ctx), context.Canceled)
	assert.Contains(t, buf.String(), "publish boom")
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestCancelledContextDoesNotCountAsAttempt(t *testing.T) {
	db, bg := newDB(t)
	require.NoError(t, outbox.Enqueue(bg, db, "a", nil))

	ctx, cancel := context.WithCancel(bg)
	pub := funcPublisher(func(string) error {
		if ctx.Err() == nil {
			cancel() // first call: the relay is shutting down mid-publish
			return context.Canceled
		}
		return errors.New("publish boom")
	})
	relay := outbox.NewRelay(db, pub, outbox.WithMaxAttempts(2),
		outbox.WithLogger(logger.New(io.Discard, logger.ErrorLevel)))

	_, err := relay.Process(ctx)
	require.ErrorIs(t, err, context.Canceled)

	// One genuine failure afterwards is attempt 1 of 2, so the event stays.
	_, err = relay.Process(bg)
	require.Error(t, err)
	dead, err := db.NewSelect().Model((*outbox.DeadEvent)(nil)).Count(bg)
	require.NoError(t, err)
	assert.Zero(t, dead, "the cancelled publish must not count towards WithMaxAttempts")
	assert.Len(t, pending(t, db, bg), 1)
}

func TestAttemptsSurviveARelayRestart(t *testing.T) {
	db, ctx := newDB(t)
	require.NoError(t, outbox.Enqueue(ctx, db, "poison", nil))
	fail := funcPublisher(func(string) error { return errors.New("publish boom") })
	newRelay := func() *outbox.Relay {
		return outbox.NewRelay(db, fail, outbox.WithMaxAttempts(3),
			outbox.WithLogger(logger.New(io.Discard, logger.ErrorLevel)))
	}

	// Two failures on one relay instance...
	first := newRelay()
	for range 2 {
		_, err := first.Process(ctx)
		require.Error(t, err)
	}
	var attempts []outbox.Attempt
	require.NoError(t, db.NewSelect().Model(&attempts).Scan(ctx))
	require.Len(t, attempts, 1)
	assert.Equal(t, 2, attempts[0].Attempts)

	// ...and the third on a fresh one (a restart, or another replica) is enough.
	_, err := newRelay().Process(ctx)
	require.NoError(t, err)

	dead, err := db.NewSelect().Model((*outbox.DeadEvent)(nil)).Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, dead)
	left, err := db.NewSelect().Model((*outbox.Attempt)(nil)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, left, "the counter is dropped with the dead-lettered event")
}

func TestPruneDropsOrphanAttempts(t *testing.T) {
	db, ctx := newDB(t)
	require.NoError(t, outbox.Enqueue(ctx, db, "flaky", nil))

	calls := 0
	pub := funcPublisher(func(string) error {
		calls++
		if calls == 1 {
			return errors.New("transient")
		}
		return nil
	})
	relay := outbox.NewRelay(db, pub, outbox.WithMaxAttempts(5))
	_, err := relay.Process(ctx)
	require.Error(t, err)
	n, err := relay.Process(ctx) // succeeds; its counter is now an orphan
	require.NoError(t, err)
	require.Equal(t, 1, n)

	_, err = outbox.Prune(ctx, db, time.Hour) // too recent to delete the event itself
	require.NoError(t, err)
	left, err := db.NewSelect().Model((*outbox.Attempt)(nil)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, left)
	assert.Len(t, pending(t, db, ctx), 1)
}
