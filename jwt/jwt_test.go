// Copyright 2026 Rahmad Afandi. MIT License.

package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
