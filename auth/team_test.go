// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveTeamAndRole(t *testing.T) {
	app := fiber.New()
	app.Get("/p", RequireAuth(middlewareTestSecret), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"team": ActiveTeam(c), "role": TeamRole(c)})
	})

	tok := mintToken(t, jwt.MapClaims{"sub": "1", "team": "7", "role": "admin"}, time.Hour)
	code, body := doBody(t, app, "GET", "/p", tok)
	require.Equal(t, 200, code)
	assert.Contains(t, body, `"team":"7"`)
	assert.Contains(t, body, `"role":"admin"`)
}

func TestRequireTeam(t *testing.T) {
	app := fiber.New()
	app.Get("/with", RequireAuth(middlewareTestSecret), RequireTeam(), func(c *fiber.Ctx) error { return c.SendStatus(200) })

	// has team -> 200
	tok := mintToken(t, jwt.MapClaims{"sub": "1", "team": "7"}, time.Hour)
	assert.Equal(t, 200, do(t, app, "GET", "/with", tok))

	// no team claim -> 403
	tok2 := mintToken(t, jwt.MapClaims{"sub": "1"}, time.Hour)
	assert.Equal(t, 403, do(t, app, "GET", "/with", tok2))

	// no auth at all -> 401
	assert.Equal(t, 401, do(t, app, "GET", "/with", ""))
}

func TestRequireRole(t *testing.T) {
	app := fiber.New()
	app.Get("/admin", RequireAuth(middlewareTestSecret), RequireRole("owner", "admin"), func(c *fiber.Ctx) error { return c.SendStatus(200) })

	owner := mintToken(t, jwt.MapClaims{"sub": "1", "team": "7", "role": "owner"}, time.Hour)
	assert.Equal(t, 200, do(t, app, "GET", "/admin", owner))

	member := mintToken(t, jwt.MapClaims{"sub": "1", "team": "7", "role": "member"}, time.Hour)
	assert.Equal(t, 403, do(t, app, "GET", "/admin", member))

	assert.Equal(t, 401, do(t, app, "GET", "/admin", ""))
}
