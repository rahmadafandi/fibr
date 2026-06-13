// Copyright 2026 Rahmad Afandi. MIT License.

package lock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// releaseScript deletes the key only if it still holds this owner's token.
// Returns 1 if deleted, 0 otherwise.
var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
else
	return 0
end`)

// extendScript renews the key's TTL only if it still holds this owner's token.
// Returns 1 if renewed, 0 otherwise.
var extendScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
	return 0
end`)

// Lock is a held lock handle. It carries the owner token so that Release and
// Extend act only when this handle still owns the key.
type Lock struct {
	client redis.UniversalClient
	key    string
	token  string
}

// Token returns the random owner token identifying this lock.
func (l *Lock) Token() string { return l.token }

// Release deletes the lock if it is still owned by this handle. It returns nil
// when the key was deleted, or ErrNotHeld when the lock was already gone or is
// owned by another holder. Release is safe to call in a defer.
func (l *Lock) Release(ctx context.Context) error {
	n, err := releaseScript.Run(ctx, l.client, []string{l.key}, l.token).Int64()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotHeld
	}
	return nil
}

// Extend renews the lock's TTL if it is still owned by this handle. It returns
// ErrNotHeld when the lock has been lost.
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	n, err := extendScript.Run(ctx, l.client, []string{l.key}, l.token, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotHeld
	}
	return nil
}
