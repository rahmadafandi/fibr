// Copyright 2026 Rahmad Afandi. MIT License.

package rediskit_test

import (
	"context"

	"github.com/rahmadafandi/fibr/v2/rediskit"
	goredis "github.com/redis/go-redis/v9"
)

// Publish an event on one instance; every instance subscribed to the channel
// runs its handler with the decoded payload (e.g. to bust a local cache).
func ExampleSubscribe() {
	r := rediskit.New(goredis.NewClient(&goredis.Options{Addr: "localhost:6379"}))
	ctx := context.Background()

	type invalidate struct {
		Key string `json:"key"`
	}

	sub, err := rediskit.Subscribe[invalidate](ctx, r, "cache:invalidate", func(_ context.Context, e invalidate) error {
		// drop e.Key from a local cache here
		return nil
	})
	if err != nil {
		return
	}
	defer sub.Close()

	_ = r.Publish(ctx, "cache:invalidate", invalidate{Key: "user:1"})
}
