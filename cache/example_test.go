// Copyright 2026 Rahmad Afandi. MIT License.

package cache_test

import (
	"context"
	"fmt"

	"github.com/rahmadafandi/fibr/cache"
)

// A typed in-memory cache with load-through. GetOrLoad runs the loader once on a
// miss (deduplicating concurrent callers) and serves the cached value after.
func ExampleCache() {
	type User struct{ Name string }

	c := cache.New[User](cache.WithMaxSize(1000))
	ctx := context.Background()

	load := func() (User, error) {
		fmt.Println("loading from db")
		return User{Name: "ada"}, nil
	}

	u, _ := c.GetOrLoad(ctx, "user:1", load) // miss -> loads
	fmt.Println(u.Name)
	u, _ = c.GetOrLoad(ctx, "user:1", load) // hit -> no load
	fmt.Println(u.Name)
	// Output:
	// loading from db
	// ada
	// ada
}
