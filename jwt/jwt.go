// Copyright 2026 Rahmad Afandi. MIT License.

// Package jwt generates and validates JSON Web Tokens.
package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

type (
	// MapClaims is re-exported from golang-jwt so callers need not import that
	// package directly.
	MapClaims = jwt.MapClaims
	// Token is re-exported from golang-jwt so callers need not import that
	// package directly.
	Token = jwt.Token
)

// ErrEmptySecret is returned when a token is generated or validated with an
// empty secret, which would make every token forgeable.
var ErrEmptySecret = errors.New("jwt: empty secret")

// GenerateToken generates a new JWT token. The token carries no "exp" claim
// unless claims already has one; prefer GenerateTokenWithExpiry.
func GenerateToken(claims MapClaims, secret string) (string, error) {
	if secret == "" {
		return "", ErrEmptySecret
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateTokenWithExpiry generates a new JWT token that expires after ttl.
// It writes the "exp" claim into the provided claims map, overwriting any
// existing "exp" value.
func GenerateTokenWithExpiry(claims MapClaims, secret string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", ErrEmptySecret
	}
	if claims == nil {
		claims = MapClaims{}
	}
	claims["exp"] = time.Now().Add(ttl).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken validates a JWT token.
func ValidateToken(tokenString string, secret string) (*Token, error) {
	if secret == "" {
		return nil, ErrEmptySecret
	}
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

// ExtractClaimsFromJwt extracts the MapClaims from a parsed JWT token.
// It returns an error if the token's Claims field is not a MapClaims value.
func ExtractClaimsFromJwt(jwtToken *Token) (MapClaims, error) {
	claims, ok := jwtToken.Claims.(MapClaims)
	if !ok {
		return nil, errors.New("jwt: invalid claims")
	}
	return claims, nil
}

// Claims gets claims from a JWT token in the fiber context or set claims to the fiber context.
func Claims(c fiber.Ctx, localKey string, claims ...MapClaims) (MapClaims, error) {
	if len(claims) > 0 {
		c.Locals(localKey, claims[0])

		return claims[0], nil
	}

	claimsData, ok := c.Locals(localKey).(MapClaims)
	if !ok {
		return MapClaims{}, errors.New("jwt: invalid claims")
	}

	return claimsData, nil
}
