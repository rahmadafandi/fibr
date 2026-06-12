// Copyright 2026 Rahmad Afandi. MIT License.

package sse_test

import (
	"bufio"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/sse"
	"github.com/stretchr/testify/require"
)

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(bufio.NewReader(r))
	require.NoError(t, err)
	return string(b)
}

func TestStreamFraming(t *testing.T) {
	app := fiber.New()
	app.Get("/events", sse.Handler(func(c *fiber.Ctx, s *sse.Stream) {
		_ = s.Send("tick", map[string]int{"n": 1})
		_ = s.Comment("keepalive")
		_ = s.SendRaw("msg", "hello")
	}))

	resp, err := app.Test(httptest.NewRequest("GET", "/events", nil), -1)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

	body := readAll(t, resp.Body)
	require.Contains(t, body, "event: tick\n")
	require.Contains(t, body, `data: {"n":1}`+"\n")
	require.Contains(t, body, ": keepalive\n")
	require.Contains(t, body, "event: msg\n")
	require.Contains(t, body, "data: hello\n")
}

func TestMultilineData(t *testing.T) {
	app := fiber.New()
	app.Get("/e", sse.Handler(func(c *fiber.Ctx, s *sse.Stream) {
		_ = s.SendRaw("x", "line1\nline2")
	}))
	resp, err := app.Test(httptest.NewRequest("GET", "/e", nil), -1)
	require.NoError(t, err)
	body := readAll(t, resp.Body)
	require.Contains(t, body, "data: line1\ndata: line2\n")
}
