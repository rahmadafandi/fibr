// Copyright 2026 Rahmad Afandi. MIT License.

package jwt

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWT(t *testing.T) {
	secret := "secret"
	claims := MapClaims{
		"name": "test",
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	}

	t.Run("GenerateToken", func(t *testing.T) {
		token, err := GenerateToken(claims, secret)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("ValidateToken", func(t *testing.T) {
		token, err := GenerateToken(claims, secret)
		assert.NoError(t, err)

		validToken, err := ValidateToken(token, secret)
		assert.NoError(t, err)
		assert.True(t, validToken.Valid)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		_, err := ValidateToken("invalid", secret)
		assert.Error(t, err)
	})

	t.Run("GenerateTokenWithExpiry", func(t *testing.T) {
		token, err := GenerateTokenWithExpiry(MapClaims{"name": "test"}, secret, time.Hour)
		assert.NoError(t, err)
		valid, err := ValidateToken(token, secret)
		assert.NoError(t, err)
		assert.True(t, valid.Valid)
	})

	t.Run("ExpiredTokenRejected", func(t *testing.T) {
		token, err := GenerateTokenWithExpiry(MapClaims{"name": "test"}, secret, -time.Hour)
		assert.NoError(t, err)
		_, err = ValidateToken(token, secret)
		assert.Error(t, err)
	})
}

func TestEmptySecretRejected(t *testing.T) {
	_, err := GenerateToken(MapClaims{"sub": "1"}, "")
	assert.ErrorIs(t, err, ErrEmptySecret)
	_, err = GenerateTokenWithExpiry(MapClaims{"sub": "1"}, "", time.Minute)
	assert.ErrorIs(t, err, ErrEmptySecret)

	// A token signed with an empty key must not validate against an empty secret.
	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "admin"}).SignedString([]byte(""))
	require.NoError(t, err)
	_, err = ValidateToken(forged, "")
	assert.ErrorIs(t, err, ErrEmptySecret)
}

func TestValidateTokenRejectsTamperedAndExpired(t *testing.T) {
	tok, err := GenerateTokenWithExpiry(nil, "secret", -time.Minute)
	require.NoError(t, err)
	_, err = ValidateToken(tok, "secret")
	assert.Error(t, err, "expired token")

	tok, err = GenerateTokenWithExpiry(MapClaims{"sub": "1"}, "secret", time.Minute)
	require.NoError(t, err)
	_, err = ValidateToken(tok, "other-secret")
	assert.Error(t, err, "wrong secret")

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "1"}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = ValidateToken(none, "secret")
	assert.Error(t, err, "alg none")
}

func TestExtractClaimsAndLocals(t *testing.T) {
	_, err := ExtractClaimsFromJwt(&Token{Claims: jwt.RegisteredClaims{}})
	assert.Error(t, err)

	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		_, err := Claims(c, "claims")
		assert.Error(t, err, "nothing stored yet")
		_, err = Claims(c, "claims", MapClaims{"sub": "7"})
		assert.NoError(t, err)
		got, err := Claims(c, "claims")
		assert.NoError(t, err)
		assert.Equal(t, "7", got["sub"])
		return nil
	})
	_, err = app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
}
