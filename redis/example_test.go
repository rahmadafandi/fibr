// Copyright 2026 Rahmad Afandi. MIT License.

package redis_test

import (
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	fibrredis "github.com/rahmadafandi/fibr/redis"
	goredis "github.com/redis/go-redis/v9"
)

// NewStorage adapts a go-redis client to fiber.Storage, e.g. for a
// multi-instance rate limiter. miniredis stands in for real Redis here.
func ExampleNewStorage() {
	mr, _ := miniredis.Run()
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	store := fibrredis.NewStorage(client, fibrredis.WithPrefix("ratelimit:"))

	_ = store.Set("1.2.3.4", []byte("1"), time.Minute)
	v, _ := store.Get("1.2.3.4")

	fmt.Println(string(v))
	// Output:
	// 1
}
