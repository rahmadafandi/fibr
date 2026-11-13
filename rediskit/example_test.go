// Copyright 2026 Rahmad Afandi. MIT License.

package rediskit_test

import (
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/rahmadafandi/fibr/v2/rediskit"
	goredis "github.com/redis/go-redis/v9"
)

// NewStorage adapts a go-redis client to fiber.Storage, e.g. for a
// multi-instance rate limiter. miniredis stands in for real Redis here.
func ExampleNewStorage() {
	mr, _ := miniredis.Run()
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	store := rediskit.NewStorage(client, rediskit.WithPrefix("ratelimit:"))

	_ = store.Set("1.2.3.4", []byte("1"), time.Minute)
	v, _ := store.Get("1.2.3.4")

	fmt.Println(string(v))
	// Output:
	// 1
}
