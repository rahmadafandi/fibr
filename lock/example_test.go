// Copyright 2026 Rahmad Afandi. MIT License.

package lock_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/rahmadafandi/fibr/lock"
	"github.com/redis/go-redis/v9"
)

// Do runs a unit of work on at most one replica. Here two Lockers (standing in
// for two replicas sharing one Redis) contend for the same periodic task: while
// replica A holds the lock and runs the work, replica B fires concurrently,
// fails to acquire, and skips with ErrNotAcquired.
func ExampleLocker_Do() {
	mr, _ := miniredis.Run()
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	replicaA := lock.New(client)
	replicaB := lock.New(client)
	ctx := context.Background()
	const key = "cron:nightly-cleanup"

	tryB := func() {
		err := replicaB.Do(ctx, key, 30*time.Second, func() error {
			fmt.Println("running on B")
			return nil
		})
		if errors.Is(err, lock.ErrNotAcquired) {
			fmt.Println("skipped on B")
		}
	}

	_ = replicaA.Do(ctx, key, 30*time.Second, func() error {
		fmt.Println("running on A")
		// B fires while A still holds the lock.
		tryB()
		return nil
	})
	// Output:
	// running on A
	// skipped on B
}
