// Copyright 2026 Rahmad Afandi. MIT License.

package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/rahmadafandi/fibr/v2/fibrtest"
	"github.com/rahmadafandi/fibr/v2/lock"
	"github.com/rahmadafandi/fibr/v2/outbox"
)

const benchBatch = 100

func benchProcess(b *testing.B, opts ...outbox.RelayOption) {
	b.Helper()
	db := fibrtest.NewDB(b)
	ctx := context.Background()
	if err := outbox.Migrate(ctx, db); err != nil {
		b.Fatal(err)
	}
	relay := outbox.NewRelay(db, funcPublisher(func(string) error { return nil }),
		append([]outbox.RelayOption{outbox.WithBatchSize(benchBatch)}, opts...)...)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		for range benchBatch {
			if err := outbox.Enqueue(ctx, db, "topic", nil); err != nil {
				b.Fatal(err)
			}
		}
		b.StartTimer()
		n, err := relay.Process(ctx)
		if err != nil || n != benchBatch {
			b.Fatalf("Process = %d, %v", n, err)
		}
	}
}

// BenchmarkProcessBatch publishes a batch of 100 events with a no-op publisher:
// what remains is the relay's own bookkeeping (one UPDATE per event).
func BenchmarkProcessBatch(b *testing.B) { benchProcess(b) }

// BenchmarkProcessBatchWithLock adds the relay lock, which is renewed before
// every event (one Redis round-trip each).
func BenchmarkProcessBatchWithLock(b *testing.B) {
	_, client := fibrtest.NewRedis(b)
	benchProcess(b, outbox.WithLock(lock.New(client), "bench:relay", time.Minute))
}
