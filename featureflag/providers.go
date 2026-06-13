// Copyright 2026 Rahmad Afandi. MIT License.

package featureflag

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"slices"

	"github.com/redis/go-redis/v9"
)

// staticProvider is a fixed on/off map.
type staticProvider map[string]bool

// Static returns a provider backed by a fixed on/off map. Unknown flags are off.
func Static(m map[string]bool) Provider {
	cp := make(staticProvider, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func (s staticProvider) Enabled(_ context.Context, flag string, _ Eval) bool {
	return s[flag]
}

// Rule targets a single flag.
type Rule struct {
	Enabled    bool     `json:"enabled"`    // fallback when no targeting matches
	Percentage int      `json:"percentage"` // 0-100 rollout, by stable hash(flag+UserID)
	Users      []string `json:"users"`      // allowlist (forces on)
	Groups     []string `json:"groups"`     // allowlist (forces on)
}

// eval applies the rule's precedence: user allowlist, then group allowlist, then
// percentage rollout, then the Enabled fallback.
func (r Rule) eval(flag string, e Eval) bool {
	if e.UserID != "" && slices.Contains(r.Users, e.UserID) {
		return true
	}
	for _, g := range e.Groups {
		if slices.Contains(r.Groups, g) {
			return true
		}
	}
	if r.Percentage > 0 && e.UserID != "" && bucket(flag, e.UserID) < r.Percentage {
		return true
	}
	return r.Enabled
}

// rulesProvider evaluates per-flag Rules.
type rulesProvider map[string]Rule

// Rules returns a provider backed by per-flag Rules. Unknown flags are off.
func Rules(m map[string]Rule) Provider {
	cp := make(rulesProvider, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func (rp rulesProvider) Enabled(_ context.Context, flag string, e Eval) bool {
	r, ok := rp[flag]
	if !ok {
		return false
	}
	return r.eval(flag, e)
}

// redisProvider reads each flag from a Redis key.
type redisProvider struct {
	client redis.UniversalClient
	prefix string
}

// Redis returns a provider that reads each flag from key prefix+flag. The value
// is "true"/"false" or a JSON Rule. A missing key or any error is off. This
// allows toggling flags live without a redeploy.
func Redis(client redis.UniversalClient, prefix string) Provider {
	return &redisProvider{client: client, prefix: prefix}
}

func (p *redisProvider) Enabled(ctx context.Context, flag string, e Eval) bool {
	val, err := p.client.Get(ctx, p.prefix+flag).Result()
	if err != nil {
		return false
	}
	switch val {
	case "true":
		return true
	case "false":
		return false
	}
	var r Rule
	if json.Unmarshal([]byte(val), &r) != nil {
		return false
	}
	return r.eval(flag, e)
}

// bucket maps a (flag, userID) pair to a stable value in [0,100), so a user's
// rollout membership does not change between checks.
func bucket(flag, userID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(flag + ":" + userID))
	return int(h.Sum32() % 100)
}
