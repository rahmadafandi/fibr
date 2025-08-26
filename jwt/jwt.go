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
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type MapClaims jwt.MapClaims

// GenerateToken generates a new JWT token.
func GenerateToken(claims jwt.MapClaims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken validates a JWT token.
func ValidateToken(tokenString string, secret string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

func ExtractClaimsFromJwt(jwtToken *jwt.Token) (jwt.MapClaims, error) {
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

// Claims gets claims from a JWT token in the fiber context or set claims to the fiber context.
func Claims(c *fiber.Ctx, localKey string, claims ...jwt.MapClaims) (jwt.MapClaims, error) {
	if len(claims) > 0 {
		c.Locals(localKey, claims[0])

		return claims[0], nil
	}

	claimsData, ok := c.Locals(localKey).(jwt.MapClaims)
	if !ok {
		return jwt.MapClaims{}, errors.New("invalid claims")
	}

	return claimsData, nil
}
