// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const middlewareTestSecret = "test-secret"

func mintToken(t *testing.T, claims jwt.MapClaims, ttl time.Duration) string {
	t.Helper()
	tok, err := jwt.GenerateTokenWithExpiry(claims, middlewareTestSecret, ttl)
	require.NoError(t, err)
	return tok
}

func do(t *testing.T, app *fiber.App, method, path, bearer string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp.StatusCode
}

func doBody(t *testing.T, app *fiber.App, method, path, bearer string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestRequireAuth(t *testing.T) {
	app := fiber.New()
	app.Get("/p", RequireAuth(middlewareTestSecret), func(c *fiber.Ctx) error {
		return c.SendString(Subject(c))
	})

	tok := mintToken(t, jwt.MapClaims{"sub": "u1"}, time.Hour)
	assert.Equal(t, 200, do(t, app, "GET", "/p", tok))
	code, body := doBody(t, app, "GET", "/p", tok)
	assert.Equal(t, 200, code)
	assert.Equal(t, "u1", body)
	assert.Equal(t, 401, do(t, app, "GET", "/p", ""))
	assert.Equal(t, 401, do(t, app, "GET", "/p", "garbage.token.here"))
	exp := mintToken(t, jwt.MapClaims{"sub": "u1"}, -time.Hour)
	assert.Equal(t, 401, do(t, app, "GET", "/p", exp))
}

func TestOptional(t *testing.T) {
	app := fiber.New()
	app.Get("/o", Optional(middlewareTestSecret), func(c *fiber.Ctx) error {
		if _, ok := Claims(c); ok {
			return c.SendString("authed")
		}
		return c.SendString("anon")
	})
	assert.Equal(t, 200, do(t, app, "GET", "/o", ""))
	assert.Equal(t, 200, do(t, app, "GET", "/o", "garbage"))
	tok := mintToken(t, jwt.MapClaims{"sub": "u1"}, time.Hour)
	code, body := doBody(t, app, "GET", "/o", tok)
	assert.Equal(t, 200, code)
	assert.Equal(t, "authed", body)
}

func TestRequireScope(t *testing.T) {
	app := fiber.New()
	app.Get("/admin", RequireAuth(middlewareTestSecret), RequireScope("admin"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	tok := mintToken(t, jwt.MapClaims{"sub": "u1", "scopes": []string{"user", "admin"}}, time.Hour)
	assert.Equal(t, 200, do(t, app, "GET", "/admin", tok))
	tok2 := mintToken(t, jwt.MapClaims{"sub": "u1", "scopes": []string{"user"}}, time.Hour)
	assert.Equal(t, 403, do(t, app, "GET", "/admin", tok2))
}

func TestRequireScopeNoClaims(t *testing.T) {
	app := fiber.New()
	app.Get("/x", RequireScope("admin"), func(c *fiber.Ctx) error { return c.SendString("ok") })
	assert.Equal(t, 401, do(t, app, "GET", "/x", ""))
}

func TestScopesNormalization(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, toStringSlice([]interface{}{"a", "b"}))
	assert.Equal(t, []string{"a"}, toStringSlice([]string{"a"}))
	assert.Nil(t, toStringSlice(nil))
	assert.Nil(t, toStringSlice("nope"))
}

func TestHasScopeAfterRoundTrip(t *testing.T) {
	app := fiber.New()
	app.Get("/r", RequireAuth(middlewareTestSecret), func(c *fiber.Ctx) error {
		assert.True(t, HasScope(c, "admin"))
		assert.False(t, HasScope(c, "nope"))
		return c.SendString("ok")
	})
	tok := mintToken(t, jwt.MapClaims{"sub": "u1", "scopes": []string{"admin"}}, time.Hour)
	assert.Equal(t, 200, do(t, app, "GET", "/r", tok))
}

func TestRequireAuthBlocklist(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	iss := NewIssuer(testSecret, store)
	pair, err := iss.Issue(ctx, jwt.MapClaims{"sub": "5", "scopes": []string{"user"}})
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/p", RequireAuth(testSecret, WithBlocklist(store)), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/p", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	ac := mustClaims(t, pair.AccessToken)
	require.NoError(t, store.Block(ctx, ac["jti"].(string), time.Hour))
	req2 := httptest.NewRequest("GET", "/p", nil)
	req2.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnauthorized, resp2.StatusCode)
}

func TestRequireAuthRejectsRefreshToken(t *testing.T) {
	iss := NewIssuer(testSecret, NewMemoryStore())
	pair, err := iss.Issue(context.Background(), jwt.MapClaims{"sub": "1"})
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/p", RequireAuth(testSecret), func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/p", nil)
	req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestOptionalBlocklistProceeds(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	iss := NewIssuer(testSecret, store)
	pair, err := iss.Issue(ctx, jwt.MapClaims{"sub": "8"})
	require.NoError(t, err)
	ac := mustClaims(t, pair.AccessToken)
	require.NoError(t, store.Block(ctx, ac["jti"].(string), time.Hour))

	app := fiber.New()
	app.Get("/p", Optional(testSecret, WithBlocklist(store)), func(c *fiber.Ctx) error {
		if _, ok := Claims(c); ok {
			return c.SendString("authed")
		}
		return c.SendString("anon")
	})

	req := httptest.NewRequest("GET", "/p", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, "anon", string(body))
}

// errStore is a TokenStore whose IsBlocked always errors, to exercise the
// fail-closed path in RequireAuth.
type errStore struct{ *MemoryStore }

func (errStore) IsBlocked(context.Context, string) (bool, error) {
	return false, errors.New("store down")
}

func TestRequireAuthBlocklistFailsClosed(t *testing.T) {
	iss := NewIssuer(testSecret, NewMemoryStore())
	pair, err := iss.Issue(context.Background(), jwt.MapClaims{"sub": "9"})
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/p", RequireAuth(testSecret, WithBlocklist(errStore{})), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/p", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode) // store error -> 401
}
