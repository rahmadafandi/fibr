// Copyright 2026 Rahmad Afandi. MIT License.

package fibrtest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/rahmadafandi/fibr/fibrtest"
	"github.com/rahmadafandi/fibr/jwt"
)

// recordTB is a stub TB that records whether Fatalf fired, so fatal paths can be
// tested without aborting the real test.
type recordTB struct {
	failed bool
	msg    string
}

func (r *recordTB) Helper() {}
func (r *recordTB) Fatalf(format string, args ...any) {
	r.failed = true
	r.msg = fmt.Sprintf(format, args...)
}

func testApp() *fiber.App {
	app := fiber.New()
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"msg": "pong"})
	})
	app.Post("/echo", func(c *fiber.Ctx) error {
		var in map[string]any
		if err := c.BodyParser(&in); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		return c.Status(fiber.StatusCreated).JSON(in)
	})
	app.Get("/auth", func(c *fiber.Ctx) error {
		if c.Get("Authorization") == "" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.JSON(fiber.Map{"auth": c.Get("Authorization")})
	})
	app.Get("/q", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"k": c.Query("k")})
	})
	return app
}

func TestGetJSON(t *testing.T) {
	c := fibrtest.New(t, testApp())
	var out struct {
		Msg string `json:"msg"`
	}
	c.Get("/ping").ExpectStatus(200).JSON(&out)
	assert.Equal(t, "pong", out.Msg)
}

func TestPostEcho(t *testing.T) {
	c := fibrtest.New(t, testApp())
	var out map[string]any
	c.Post("/echo", map[string]string{"name": "x"}).ExpectStatus(201).JSON(&out)
	assert.Equal(t, "x", out["name"])
}

func TestWithBearer(t *testing.T) {
	c := fibrtest.New(t, testApp()).WithBearer("tok123")
	var out struct {
		Auth string `json:"auth"`
	}
	c.Get("/auth").ExpectStatus(200).JSON(&out)
	assert.Equal(t, "Bearer tok123", out.Auth)
}

func TestRequestBearerOverridesAndDefaultUnchanged(t *testing.T) {
	base := fibrtest.New(t, testApp())
	// No default bearer -> 401.
	base.Get("/auth").ExpectStatus(401)
	// Per-request bearer -> 200.
	base.Request("GET", "/auth").Bearer("abc").Do().ExpectStatus(200)
}

func TestQuery(t *testing.T) {
	c := fibrtest.New(t, testApp())
	var out struct {
		K string `json:"k"`
	}
	c.Request("GET", "/q").Query("k", "v").Do().ExpectStatus(200).JSON(&out)
	assert.Equal(t, "v", out.K)
}

func TestExpectStatusMismatchCallsFatalf(t *testing.T) {
	rt := &recordTB{}
	c := fibrtest.New(rt, testApp())
	c.Get("/ping").ExpectStatus(404)
	assert.True(t, rt.failed)
	assert.Contains(t, rt.msg, "expected status 404")
}

func TestJSONDecodeErrorCallsFatalf(t *testing.T) {
	rt := &recordTB{}
	c := fibrtest.New(rt, testApp())
	var wrong int // body is an object, cannot decode into int
	c.Get("/ping").JSON(&wrong)
	assert.True(t, rt.failed)
}

func TestNewDB(t *testing.T) {
	db := fibrtest.NewDB(t)

	type widget struct {
		bun.BaseModel `bun:"table:widgets,alias:w"`
		ID            int64  `bun:"id,pk,autoincrement"`
		Name          string `bun:"name"`
	}
	ctx := context.Background()
	_, err := db.NewCreateTable().Model((*widget)(nil)).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&widget{Name: "gear"}).Exec(ctx)
	require.NoError(t, err)

	var got widget
	require.NoError(t, db.NewSelect().Model(&got).Limit(1).Scan(ctx))
	assert.Equal(t, "gear", got.Name)
}

func TestToken(t *testing.T) {
	const secret = "s3cr3t"
	tok := fibrtest.Token(t, secret, jwt.MapClaims{"sub": "u1"})
	require.NotEmpty(t, tok)

	parsed, err := jwt.ValidateToken(tok, secret)
	require.NoError(t, err)
	claims, err := jwt.ExtractClaimsFromJwt(parsed)
	require.NoError(t, err)
	assert.Equal(t, "u1", claims["sub"])
}
