// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"context"
	"time"
)

// TokenStore tracks revoked token ids (blocklist) and the active refresh token
// per family, enabling logout, revocation, and refresh-token rotation with
// reuse detection. Implementations must be safe for concurrent use.
type TokenStore interface {
	// Block marks jti revoked until ttl elapses.
	Block(ctx context.Context, jti string, ttl time.Duration) error
	// IsBlocked reports whether jti is currently revoked.
	IsBlocked(ctx context.Context, jti string) (bool, error)
	// SetFamily records jti as the active refresh token for family fid until ttl.
	SetFamily(ctx context.Context, fid, jti string, ttl time.Duration) error
	// Family returns the active refresh jti for fid; ok is false when the family
	// is absent or revoked.
	Family(ctx context.Context, fid string) (jti string, ok bool, err error)
	// RevokeFamily removes the family record, invalidating all its tokens.
	RevokeFamily(ctx context.Context, fid string) error
}
