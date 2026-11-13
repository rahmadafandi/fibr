// Copyright 2026 Rahmad Afandi. MIT License.

package cache

import (
	"strconv"
	"testing"
)

const benchKeys = 1024

func benchCache(b *testing.B) (*Cache[int], []string) {
	b.Helper()
	c := New[int](WithMaxSize(benchKeys))
	b.Cleanup(c.Close)
	keys := make([]string, benchKeys)
	for i := range keys {
		keys[i] = "key:" + strconv.Itoa(i)
		c.Set(keys[i], i)
	}
	return c, keys
}

func BenchmarkGet(b *testing.B) {
	c, keys := benchCache(b)
	b.ReportAllocs()
	b.ResetTimer()
	i := 0
	for b.Loop() {
		c.Get(keys[i%benchKeys])
		i++
	}
}

// BenchmarkGetParallel shows how reads scale across cores. Get takes the one
// mutex (it moves the entry in the LRU list), so readers serialize.
func BenchmarkGetParallel(b *testing.B) {
	c, keys := benchCache(b)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(keys[i%benchKeys])
			i++
		}
	})
}

func BenchmarkSetParallel(b *testing.B) {
	c, keys := benchCache(b)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(keys[i%benchKeys], i)
			i++
		}
	})
}
