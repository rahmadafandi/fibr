// Copyright 2026 Rahmad Afandi. MIT License.

package ratelimit_test

import (
	"context"
	"fmt"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/rahmadafandi/fibr/ratelimit"
)

// A token bucket of 2 with a slow refill: the first two requests pass, the third
// is denied.
func ExampleLimiter_Allow() {
	mr, _ := miniredis.Run()
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	l := ratelimit.New(client)
	rule := ratelimit.Rule{Capacity: 2, RefillPerSec: 1}
	ctx := context.Background()

	for range 3 {
		res, _ := l.Allow(ctx, "user:1", rule, 1)
		fmt.Println(res.Allowed)
	}
	// Output:
	// true
	// true
	// false
}
