// Copyright 2026 Rahmad Afandi. MIT License.

package webhook_test

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/apierror"
	"github.com/rahmadafandi/fibr/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareValid(t *testing.T) {
	secret := "secret"
	body := `{"event":"ok"}`

	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Use(webhook.Middleware(webhook.MiddlewareConfig{Secret: secret}))
	app.Post("/hook", func(c *fiber.Ctx) error { return c.SendString("handled") })

	req := httptest.NewRequest("POST", "/hook", strings.NewReader(body))
	req.Header.Set(webhook.DefaultHeader, webhook.Sign([]byte(body), secret, time.Now()))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	b, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "handled", string(b))
}

func TestMiddlewareRejectsBadSignature(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Use(webhook.Middleware(webhook.MiddlewareConfig{Secret: "secret"}))
	app.Post("/hook", func(c *fiber.Ctx) error { return c.SendString("handled") })

	req := httptest.NewRequest("POST", "/hook", strings.NewReader(`{"event":"ok"}`))
	req.Header.Set(webhook.DefaultHeader, "t=1,v1=deadbeef")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}
