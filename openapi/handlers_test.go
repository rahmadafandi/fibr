// Copyright 2026 Rahmad Afandi. MIT License.

package openapi

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func buildApp(s *Spec) *fiber.App {
	app := fiber.New()
	app.Get("/openapi.json", s.SpecHandler())
	app.Get("/docs", s.UIHandler("/openapi.json"))
	return app
}

func TestSpecHandlerServesJSON(t *testing.T) {
	s := New(Info{Title: "API", Version: "1.0.0"})
	s.Register("GET", "/ping", Op{Summary: "ping"})
	app := buildApp(s)

	resp, err := app.Test(httptest.NewRequest("GET", "/openapi.json", nil))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	body, _ := io.ReadAll(resp.Body)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(body, &doc))
	require.Equal(t, "3.0.3", doc["openapi"])
}

func TestSpecHandlerCaches(t *testing.T) {
	s := New(Info{Title: "API", Version: "1.0.0"})
	b1, err := s.bytes()
	require.NoError(t, err)
	b2, err := s.bytes()
	require.NoError(t, err)
	require.Same(t, &b1[0], &b2[0]) // same backing array -> cached
}

func TestUIHandlerServesHTML(t *testing.T) {
	s := New(Info{Title: "API", Version: "1.0.0"})
	app := buildApp(s)

	resp, err := app.Test(httptest.NewRequest("GET", "/docs", nil))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	require.True(t, strings.Contains(html, "swagger-ui"))
	require.True(t, strings.Contains(html, "/openapi.json"))
}
