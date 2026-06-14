// Copyright 2026 Rahmad Afandi. MIT License.

// Package ratelimit is a Redis-backed token-bucket rate limiter with per-key
// buckets, cost-per-request, and a Fiber middleware.
package ratelimit

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "ratelimit:"

// tokenBucket atomically refills a bucket by elapsed*rate, consumes cost tokens
// if available, and persists {tokens, ts}. ARGV: capacity, rate(tokens/ms),
// now(ms), cost, ttl(ms). Returns {allowed, floor(tokens)}.
var tokenBucket = redis.NewScript(`
local cap  = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now  = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local ttl  = tonumber(ARGV[5])
local d = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(d[1])
local ts = tonumber(d[2])
if tokens == nil then tokens = cap; ts = now end
local elapsed = now - ts
if elapsed < 0 then elapsed = 0 end
tokens = math.min(cap, tokens + elapsed * rate)
local allowed = 0
if tokens >= cost then allowed = 1; tokens = tokens - cost end
redis.call('HMSET', KEYS[1], 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', KEYS[1], ttl)
return {allowed, math.floor(tokens)}`)

// Limiter is a token-bucket rate limiter backed by Redis.
type Limiter struct {
	client redis.UniversalClient
	now    func() time.Time
}

// New returns a Limiter using the given Redis client.
func New(client redis.UniversalClient) *Limiter {
	return &Limiter{client: client, now: time.Now}
}

// Rule defines a bucket: Capacity tokens, refilled at RefillPerSec tokens/sec.
type Rule struct {
	Capacity     int
	RefillPerSec float64
}

// Result is the outcome of an Allow call.
type Result struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration // > 0 only when not allowed
}

// Allow attempts to consume cost tokens for key under rule.
func (l *Limiter) Allow(ctx context.Context, key string, rule Rule, cost int) (Result, error) {
	if cost < 1 {
		cost = 1
	}
	ratePerMs := rule.RefillPerSec / 1000.0
	nowMs := l.now().UnixMilli()
	ttlMs := bucketTTLMs(rule)

	res, err := tokenBucket.Run(ctx, l.client, []string{keyPrefix + key},
		rule.Capacity, ratePerMs, nowMs, cost, ttlMs).Int64Slice()
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit: eval: %w", err)
	}

	allowed := res[0] == 1
	remaining := int(res[1])
	out := Result{Allowed: allowed, Remaining: remaining}
	if !allowed {
		out.RetryAfter = retryAfter(cost, remaining, rule.RefillPerSec)
	}
	return out, nil
}

// bucketTTLMs is how long an idle bucket lives: the time to refill it fully,
// at least one second.
func bucketTTLMs(rule Rule) int64 {
	if rule.RefillPerSec <= 0 {
		return 1000
	}
	ms := int64(math.Ceil(float64(rule.Capacity)/rule.RefillPerSec)) * 1000
	if ms < 1000 {
		ms = 1000
	}
	return ms
}

// retryAfter is how long until enough tokens to satisfy cost are available.
func retryAfter(cost, remaining int, refillPerSec float64) time.Duration {
	if refillPerSec <= 0 {
		return time.Second
	}
	need := float64(cost - remaining)
	if need <= 0 {
		return time.Second
	}
	secs := math.Ceil(need / refillPerSec)
	if secs < 1 {
		secs = 1
	}
	return time.Duration(secs) * time.Second
}
