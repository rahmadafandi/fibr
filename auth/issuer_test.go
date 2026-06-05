// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/rahmadafandi/fiber-helpers/jwt"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-0123456789"

func TestIssuerIssue(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	iss := NewIssuer(testSecret, store, WithAccessTTL(15*time.Minute), WithRefreshTTL(48*time.Hour))

	pair, err := iss.Issue(ctx, jwt.MapClaims{
		"sub":    "42",
		"email":  "a@example.com",
		"scopes": []string{"user"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	require.Equal(t, "Bearer", pair.TokenType)
	require.Equal(t, int64((15 * time.Minute).Seconds()), pair.ExpiresIn)
	require.Equal(t, int64((48 * time.Hour).Seconds()), pair.RefreshExpiresIn)

	ac := mustClaims(t, pair.AccessToken)
	require.Equal(t, "access", ac["type"])
	require.NotEmpty(t, ac["jti"])
	require.Equal(t, "42", ac["sub"])
	require.Nil(t, ac["fid"])
	require.Equal(t, "a@example.com", ac["email"])                  // email propagated
	require.Equal(t, []interface{}{"user"}, ac["scopes"])           // scopes propagated (JSON-decoded form)

	rc := mustClaims(t, pair.RefreshToken)
	require.Equal(t, "refresh", rc["type"])
	require.NotEmpty(t, rc["fid"])
	require.NotEmpty(t, rc["jti"])

	active, ok, err := store.Family(ctx, rc["fid"].(string))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, rc["jti"], active)
}

func TestIssuerIssueRequiresSub(t *testing.T) {
	iss := NewIssuer(testSecret, NewMemoryStore())
	_, err := iss.Issue(context.Background(), jwt.MapClaims{"email": "x@example.com"})
	require.Error(t, err)
}

func TestIssuerDefaults(t *testing.T) {
	iss := NewIssuer(testSecret, NewMemoryStore())
	pair, err := iss.Issue(context.Background(), jwt.MapClaims{"sub": "1"})
	require.NoError(t, err)
	require.Equal(t, int64((15 * time.Minute).Seconds()), pair.ExpiresIn)
	require.Equal(t, int64((168 * time.Hour).Seconds()), pair.RefreshExpiresIn)
}

// mustClaims validates a token with testSecret and returns its claims.
func mustClaims(t *testing.T, token string) jwt.MapClaims {
	t.Helper()
	parsed, err := jwt.ValidateToken(token, testSecret)
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	claims, err := jwt.ExtractClaimsFromJwt(parsed)
	require.NoError(t, err)
	return claims
}
