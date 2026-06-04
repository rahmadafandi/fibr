// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func newTestRedis(t *testing.T) *Redis {
	t.Helper()
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return New(client)
}

func TestSetGet(t *testing.T) {
	r := newTestRedis(t)
	ctx := context.Background()

	type item struct{ Name string }
	assert.NoError(t, r.Set(ctx, "k", item{Name: "x"}, time.Minute))

	var got item
	assert.NoError(t, r.Get(ctx, "k", &got))
	assert.Equal(t, "x", got.Name)
}

func TestRememberMissThenHit(t *testing.T) {
	r := newTestRedis(t)
	ctx := context.Background()

	calls := 0
	loader := func() (string, error) {
		calls++
		return "loaded", nil
	}

	v, err := Remember(ctx, r, "key", time.Minute, loader)
	assert.NoError(t, err)
	assert.Equal(t, "loaded", v)
	assert.Equal(t, 1, calls)

	v, err = Remember(ctx, r, "key", time.Minute, loader)
	assert.NoError(t, err)
	assert.Equal(t, "loaded", v)
	assert.Equal(t, 1, calls)
}

func TestRememberLoaderError(t *testing.T) {
	r := newTestRedis(t)
	ctx := context.Background()

	_, err := Remember(ctx, r, "key", time.Minute, func() (string, error) {
		return "", assert.AnError
	})
	assert.Error(t, err)
}
