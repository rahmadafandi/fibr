// Copyright 2026 Rahmad Afandi. MIT License.

package redis_test

import (
	"context"

	fibrredis "github.com/rahmadafandi/fibr/redis"
	goredis "github.com/redis/go-redis/v9"
)

// Publish an event on one instance; every instance subscribed to the channel
// runs its handler with the decoded payload (e.g. to bust a local cache).
func ExampleSubscribe() {
	r := fibrredis.New(goredis.NewClient(&goredis.Options{Addr: "localhost:6379"}))
	ctx := context.Background()

	type invalidate struct {
		Key string `json:"key"`
	}

	sub, err := fibrredis.Subscribe[invalidate](ctx, r, "cache:invalidate", func(_ context.Context, e invalidate) error {
		// drop e.Key from a local cache here
		return nil
	})
	if err != nil {
		return
	}
	defer sub.Close()

	_ = r.Publish(ctx, "cache:invalidate", invalidate{Key: "user:1"})
}
