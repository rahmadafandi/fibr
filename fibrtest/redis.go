// Copyright 2026 Rahmad Afandi. MIT License.

package fibrtest

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// NewRedis starts an in-memory Redis server (miniredis) and returns it together
// with a go-redis client connected to it. Use the server handle to inspect or
// manipulate state (keys, TTLs, time). When t supports Cleanup (a real
// *testing.T or *testing.B), both are closed automatically at test end;
// otherwise the caller is responsible for closing them. A startup error calls
// Fatalf.
func NewRedis(t TB) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("fibrtest: start miniredis: %v", err)
		return nil, nil
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	if c, ok := t.(interface{ Cleanup(func()) }); ok {
		c.Cleanup(func() {
			_ = client.Close()
			mr.Close()
		})
	}
	return mr, client
}
