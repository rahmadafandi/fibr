// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/rahmadafandi/fibr/jwt"
)

// Errors returned by the Issuer. Callers should map both to HTTP 401;
// ErrTokenReuse additionally signals a likely stolen refresh token (the whole
// token family is revoked when it occurs).
var (
	ErrInvalidToken = errors.New("auth: invalid token")
	ErrTokenReuse   = errors.New("auth: refresh token reuse detected")
)

// TokenPair is the access+refresh token pair returned by Issue and Refresh.
type TokenPair struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int64  `json:"expires_in"`         // access token lifetime, seconds
	RefreshExpiresIn int64  `json:"refresh_expires_in"` // refresh token lifetime, seconds
}

// Issuer mints, rotates, and revokes JWT access/refresh token pairs backed by a
// TokenStore for revocation and refresh-family tracking.
type Issuer struct {
	secret     string
	store      TokenStore
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// IssuerOption configures an Issuer.
type IssuerOption func(*Issuer)

// WithAccessTTL sets the access token lifetime (default 15m).
func WithAccessTTL(d time.Duration) IssuerOption {
	return func(i *Issuer) {
		if d > 0 {
			i.accessTTL = d
		}
	}
}

// WithRefreshTTL sets the refresh token lifetime (default 7 days).
func WithRefreshTTL(d time.Duration) IssuerOption {
	return func(i *Issuer) {
		if d > 0 {
			i.refreshTTL = d
		}
	}
}

// NewIssuer creates an Issuer signing with secret and persisting revocation /
// family state in store.
func NewIssuer(secret string, store TokenStore, opts ...IssuerOption) *Issuer {
	i := &Issuer{
		secret:     secret,
		store:      store,
		accessTTL:  15 * time.Minute,
		refreshTTL: 168 * time.Hour,
	}
	for _, o := range opts {
		o(i)
	}
	return i
}

// Issue mints a fresh access+refresh pair for a new login, allocating a new
// token family. claims must contain a non-empty string "sub"; "email" and
// "scopes" are copied into both tokens when present.
func (i *Issuer) Issue(ctx context.Context, claims jwt.MapClaims) (TokenPair, error) {
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return TokenPair{}, errors.New("auth: issue: missing sub claim")
	}
	fid, err := newID()
	if err != nil {
		return TokenPair{}, err
	}
	return i.mint(ctx, claims, fid)
}

// Refresh validates a refresh token and rotates it: it returns a new pair under
// the same family with fresh ids and updates the family's active token. If the
// presented token has been superseded (reuse), it revokes the entire family and
// returns ErrTokenReuse; any other validation failure returns ErrInvalidToken.
//
// Reuse detection is sequential: once a token is rotated, presenting the old
// token again is caught by the family-pointer mismatch (the primary defense).
// The check-and-rotate is not atomic, so two Refresh calls racing on the *same*
// token may both succeed; last-write-wins on the family pointer makes the loser's
// token immediately stale, so the next use of it kills the family. If you need
// strict single-use under concurrent presentation, back the store with an atomic
// compare-and-swap implementation.
func (i *Issuer) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := i.parseValid(refreshToken)
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}
	if t, _ := claims["type"].(string); t != "refresh" {
		return TokenPair{}, ErrInvalidToken
	}
	fid, _ := claims["fid"].(string)
	jti, _ := claims["jti"].(string)
	if fid == "" || jti == "" {
		return TokenPair{}, ErrInvalidToken
	}

	active, ok, err := i.store.Family(ctx, fid)
	if err != nil {
		return TokenPair{}, err
	}
	if !ok {
		return TokenPair{}, ErrInvalidToken
	}
	if active != jti {
		_ = i.store.RevokeFamily(ctx, fid)
		return TokenPair{}, ErrTokenReuse
	}

	pair, err := i.mint(ctx, claims, fid)
	if err != nil {
		return TokenPair{}, err
	}
	if rem := remainingTTL(claims); rem > 0 {
		_ = i.store.Block(ctx, jti, rem)
	}
	return pair, nil
}

// mint builds an access+refresh pair for the given family, signs both, and
// records the new refresh jti as the family's active token.
func (i *Issuer) mint(ctx context.Context, base jwt.MapClaims, fid string) (TokenPair, error) {
	accessJti, err := newID()
	if err != nil {
		return TokenPair{}, err
	}
	refreshJti, err := newID()
	if err != nil {
		return TokenPair{}, err
	}

	access := jwt.MapClaims{"sub": base["sub"], "type": "access", "jti": accessJti}
	refresh := jwt.MapClaims{"sub": base["sub"], "type": "refresh", "fid": fid, "jti": refreshJti}
	for _, k := range []string{"email", "scopes", "team", "role"} {
		if v, ok := base[k]; ok {
			access[k] = v
			refresh[k] = v
		}
	}

	at, err := jwt.GenerateTokenWithExpiry(access, i.secret, i.accessTTL)
	if err != nil {
		return TokenPair{}, err
	}
	rt, err := jwt.GenerateTokenWithExpiry(refresh, i.secret, i.refreshTTL)
	if err != nil {
		return TokenPair{}, err
	}
	if err := i.store.SetFamily(ctx, fid, refreshJti, i.refreshTTL); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:      at,
		RefreshToken:     rt,
		TokenType:        "Bearer",
		ExpiresIn:        int64(i.accessTTL.Seconds()),
		RefreshExpiresIn: int64(i.refreshTTL.Seconds()),
	}, nil
}

// parseValid validates token with the issuer secret and returns its claims.
func (i *Issuer) parseValid(token string) (jwt.MapClaims, error) {
	parsed, err := jwt.ValidateToken(token, i.secret)
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	claims, err := jwt.ExtractClaimsFromJwt(parsed)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// newID returns a random 16-byte hex identifier.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Logout revokes a session. It blocks the access token's jti until its
// expiry and revokes the refresh token's family. Either argument may be empty;
// malformed or invalid tokens are ignored so logout is best-effort and never
// fails on junk input. It returns the first store error encountered, if any.
func (i *Issuer) Logout(ctx context.Context, accessToken, refreshToken string) error {
	var firstErr error
	if accessToken != "" {
		if claims, err := i.parseValid(accessToken); err == nil {
			if jti, _ := claims["jti"].(string); jti != "" {
				if rem := remainingTTL(claims); rem > 0 {
					if e := i.store.Block(ctx, jti, rem); e != nil && firstErr == nil {
						firstErr = e
					}
				}
			}
		}
	}
	if refreshToken != "" {
		if claims, err := i.parseValid(refreshToken); err == nil {
			if fid, _ := claims["fid"].(string); fid != "" {
				if e := i.store.RevokeFamily(ctx, fid); e != nil && firstErr == nil {
					firstErr = e
				}
			}
		}
	}
	return firstErr
}

// remainingTTL derives the time until a token's "exp" claim from now (0 if
// absent, unparseable, or already past).
func remainingTTL(claims jwt.MapClaims) time.Duration {
	var unix int64
	switch v := claims["exp"].(type) {
	case float64:
		unix = int64(v)
	case int64:
		unix = v
	case json.Number:
		unix, _ = v.Int64()
	default:
		return 0
	}
	d := time.Until(time.Unix(unix, 0))
	if d < 0 {
		return 0
	}
	return d
}
