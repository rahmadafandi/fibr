// Copyright 2026 Rahmad Afandi. MIT License.

package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
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

// GenerateToken generates a new JWT token.
func GenerateToken(claims MapClaims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateTokenWithExpiry generates a new JWT token that expires after ttl.
// It writes the "exp" claim into the provided claims map, overwriting any
// existing "exp" value.
func GenerateTokenWithExpiry(claims MapClaims, secret string, ttl time.Duration) (string, error) {
	if claims == nil {
		claims = MapClaims{}
	}
	claims["exp"] = time.Now().Add(ttl).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken validates a JWT token.
func ValidateToken(tokenString string, secret string) (*Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

// ExtractClaimsFromJwt extracts the MapClaims from a parsed JWT token.
// It returns an error if the token's Claims field is not a MapClaims value.
func ExtractClaimsFromJwt(jwtToken *Token) (MapClaims, error) {
	claims, ok := jwtToken.Claims.(MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

// Claims gets claims from a JWT token in the fiber context or set claims to the fiber context.
func Claims(c *fiber.Ctx, localKey string, claims ...MapClaims) (MapClaims, error) {
	if len(claims) > 0 {
		c.Locals(localKey, claims[0])

		return claims[0], nil
	}

	claimsData, ok := c.Locals(localKey).(MapClaims)
	if !ok {
		return MapClaims{}, errors.New("invalid claims")
	}

	return claimsData, nil
}
