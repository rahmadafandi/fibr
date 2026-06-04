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
