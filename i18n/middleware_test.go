// Copyright 2026 Rahmad Afandi. MIT License.

package i18n_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/i18n"
	"github.com/stretchr/testify/require"
)

func app(t *testing.T) *fiber.App {
	t.Helper()
	a := fiber.New()
	a.Use(i18n.Middleware(bundle(t)))
	a.Get("/hello", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"locale": i18n.Locale(c), "msg": i18n.T(c, "welcome", i18n.M{"name": "Sam"})})
	})
	return a
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

func TestQueryWins(t *testing.T) {
	a := app(t)
	req := httptest.NewRequest("GET", "/hello?lang=id", nil)
	req.Header.Set("Accept-Language", "en")
	resp, err := a.Test(req)
	require.NoError(t, err)
	body := readBody(t, resp)
	require.Contains(t, body, `"locale":"id"`)
	require.Contains(t, body, "Halo, Sam!")
}

func TestCookieOverHeader(t *testing.T) {
	a := app(t)
	req := httptest.NewRequest("GET", "/hello", nil)
	req.Header.Set("Cookie", "lang=id")
	req.Header.Set("Accept-Language", "en")
	resp, err := a.Test(req)
	require.NoError(t, err)
	require.Contains(t, readBody(t, resp), `"locale":"id"`)
}

func TestAcceptLanguage(t *testing.T) {
	a := app(t)
	req := httptest.NewRequest("GET", "/hello", nil)
	req.Header.Set("Accept-Language", "id-ID,id;q=0.9,en;q=0.8")
	resp, err := a.Test(req)
	require.NoError(t, err)
	require.Contains(t, readBody(t, resp), `"locale":"id"`)
}

func TestFallbackWhenUnknown(t *testing.T) {
	a := app(t)
	req := httptest.NewRequest("GET", "/hello?lang=fr", nil)
	resp, err := a.Test(req)
	require.NoError(t, err)
	require.Contains(t, readBody(t, resp), `"locale":"en"`)
}
