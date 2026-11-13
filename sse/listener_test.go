// Copyright 2026 Rahmad Afandi. MIT License.

package sse_test

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/v2/sse"
)

// serve runs app on a real TCP listener. app.Test does not exercise the
// streaming path the way a server does, so these tests need a socket.
func serve(t *testing.T, app *fiber.App) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() { _ = app.Shutdown() })
	return "http://" + ln.Addr().String()
}

// The stream function runs after the route handler has returned. It must still
// be able to use the fiber.Ctx it was given.
func TestHandlerCtxUsableWhileStreaming(t *testing.T) {
	app := fiber.New()
	app.Get("/events/:room", sse.Handler(func(c fiber.Ctx, s *sse.Stream) {
		for n := 0; n < 3; n++ {
			if err := s.Send("tick", fiber.Map{"room": c.Params("room"), "n": n}); err != nil {
				return
			}
			select {
			case <-c.RequestCtx().Done():
				return
			case <-time.After(10 * time.Millisecond):
			}
		}
	}))
	base := serve(t, app)

	resp, err := http.Get(base + "/events/lobby")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	var data []string
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if line := sc.Text(); strings.HasPrefix(line, "data: ") {
			data = append(data, strings.TrimPrefix(line, "data: "))
		}
	}
	require.Len(t, data, 3)
	assert.JSONEq(t, `{"room":"lobby","n":2}`, data[2])
}

// A panic in the stream function must not take the whole process down.
func TestHandlerRecoversPanicInStream(t *testing.T) {
	app := fiber.New()
	app.Get("/boom", sse.Handler(func(_ fiber.Ctx, s *sse.Stream) {
		_ = s.SendRaw("msg", "before")
		panic("stream boom")
	}))
	app.Get("/ok", func(c fiber.Ctx) error { return c.SendString("still up") })
	base := serve(t, app)

	resp, err := http.Get(base + "/boom")
	require.NoError(t, err)
	body, _ := bufio.NewReader(resp.Body).ReadString(0)
	_ = resp.Body.Close()
	assert.Contains(t, body, "data: before")

	resp, err = http.Get(base + "/ok")
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}
