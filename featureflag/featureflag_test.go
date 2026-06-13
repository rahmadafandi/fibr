// Copyright 2026 Rahmad Afandi. MIT License.

package featureflag

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticProvider(t *testing.T) {
	f := New(Static(map[string]bool{"on": true, "off": false}))
	ctx := context.Background()
	assert.True(t, f.Enabled(ctx, "on", Eval{}))
	assert.False(t, f.Enabled(ctx, "off", Eval{}))
	assert.False(t, f.Enabled(ctx, "unknown", Eval{}))
}

func TestRulesMasterAndUnknown(t *testing.T) {
	f := New(Rules(map[string]Rule{"all": {Enabled: true}}))
	ctx := context.Background()
	assert.True(t, f.Enabled(ctx, "all", Eval{UserID: "anyone"}))
	assert.False(t, f.Enabled(ctx, "missing", Eval{UserID: "anyone"}))
}

func TestRulesUserAndGroupAllowlist(t *testing.T) {
	f := New(Rules(map[string]Rule{
		"beta": {Users: []string{"u1"}, Groups: []string{"staff"}},
	}))
	ctx := context.Background()
	assert.True(t, f.Enabled(ctx, "beta", Eval{UserID: "u1"}))
	assert.True(t, f.Enabled(ctx, "beta", Eval{UserID: "u2", Groups: []string{"staff"}}))
	assert.False(t, f.Enabled(ctx, "beta", Eval{UserID: "u2", Groups: []string{"other"}}))
}

func TestRulesPercentageBounds(t *testing.T) {
	ctx := context.Background()
	none := New(Rules(map[string]Rule{"f": {Percentage: 0}}))
	all := New(Rules(map[string]Rule{"f": {Percentage: 100}}))
	for _, u := range []string{"a", "b", "c", "d"} {
		assert.False(t, none.Enabled(ctx, "f", Eval{UserID: u}))
		assert.True(t, all.Enabled(ctx, "f", Eval{UserID: u}))
	}
}

func TestRulesPercentageDeterministic(t *testing.T) {
	f := New(Rules(map[string]Rule{"f": {Percentage: 50}}))
	ctx := context.Background()
	first := f.Enabled(ctx, "f", Eval{UserID: "stable-user"})
	for range 5 {
		assert.Equal(t, first, f.Enabled(ctx, "f", Eval{UserID: "stable-user"}))
	}
}

func TestBucketRange(t *testing.T) {
	for _, u := range []string{"", "a", "longer-user-id", "x123"} {
		b := bucket("flag", u)
		assert.GreaterOrEqual(t, b, 0)
		assert.Less(t, b, 100)
	}
	assert.Equal(t, bucket("flag", "u"), bucket("flag", "u")) // stable
}

func TestRedisProvider(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "ff:on", "true", 0).Err())
	require.NoError(t, client.Set(ctx, "ff:off", "false", 0).Err())
	require.NoError(t, client.Set(ctx, "ff:rule", `{"users":["u1"]}`, 0).Err())

	f := New(Redis(client, "ff:"))
	assert.True(t, f.Enabled(ctx, "on", Eval{}))
	assert.False(t, f.Enabled(ctx, "off", Eval{}))
	assert.True(t, f.Enabled(ctx, "rule", Eval{UserID: "u1"}))
	assert.False(t, f.Enabled(ctx, "rule", Eval{UserID: "u2"}))
	assert.False(t, f.Enabled(ctx, "missing", Eval{})) // no key -> off
}
