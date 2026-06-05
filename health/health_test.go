// Copyright 2026 Rahmad Afandi. MIT License.

package health

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func doGet(t *testing.T, app *fiber.App, path string) (int, map[string]interface{}) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", path, nil))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	var m map[string]interface{}
	_ = json.Unmarshal(body, &m)
	return resp.StatusCode, m
}

func TestLivez(t *testing.T) {
	app := fiber.New()
	Register(app)
	code, body := doGet(t, app, "/livez")
	assert.Equal(t, 200, code)
	assert.Equal(t, "ok", body["status"])
}

func TestReadyzNoChecks(t *testing.T) {
	app := fiber.New()
	Register(app)
	code, body := doGet(t, app, "/readyz")
	assert.Equal(t, 200, code)
	assert.Equal(t, "ok", body["status"])
}

func TestReadyzPassingCheck(t *testing.T) {
	app := fiber.New()
	Register(app, Check("always", func(ctx context.Context) error { return nil }))
	code, body := doGet(t, app, "/readyz")
	assert.Equal(t, 200, code)
	assert.Equal(t, "ok", body["status"])
	checks := body["checks"].(map[string]interface{})
	assert.Equal(t, "ok", checks["always"])
}

func TestReadyzFailingCheck(t *testing.T) {
	app := fiber.New()
	Register(app,
		Check("good", func(ctx context.Context) error { return nil }),
		Check("bad", func(ctx context.Context) error { return assert.AnError }),
	)
	code, body := doGet(t, app, "/readyz")
	assert.Equal(t, 503, code)
	assert.Equal(t, "error", body["status"])
	checks := body["checks"].(map[string]interface{})
	assert.Equal(t, "ok", checks["good"])
	assert.NotEqual(t, "ok", checks["bad"])
}

func TestReadyzRecoversPanic(t *testing.T) {
	app := fiber.New()
	Register(app, Check("panicky", func(ctx context.Context) error { panic("boom") }))
	code, _ := doGet(t, app, "/readyz")
	assert.Equal(t, 503, code)
}

func TestRegisterAtCustomPaths(t *testing.T) {
	app := fiber.New()
	RegisterAt(app, "/alive", "/ready")
	code, _ := doGet(t, app, "/alive")
	assert.Equal(t, 200, code)
	code, _ = doGet(t, app, "/ready")
	assert.Equal(t, 200, code)
}

func TestRegisterProviderLiveChecks(t *testing.T) {
	app := fiber.New()
	checks := []NamedCheck{}
	RegisterProvider(app, func() []NamedCheck { return checks })

	// No checks yet -> ok.
	code, body := doGet(t, app, "/readyz")
	assert.Equal(t, 200, code)
	assert.Equal(t, "ok", body["status"])

	// Add a failing check AFTER registration -> reflected live.
	checks = append(checks, Check("late", func(ctx context.Context) error { return assert.AnError }))
	code, body = doGet(t, app, "/readyz")
	assert.Equal(t, 503, code)
	assert.Equal(t, "error", body["status"])

	// /livez still works.
	code, _ = doGet(t, app, "/livez")
	assert.Equal(t, 200, code)
}

func TestReadyzTimesOutSlowCheck(t *testing.T) {
	old := defaultCheckTimeout
	defaultCheckTimeout = 100 * time.Millisecond
	defer func() { defaultCheckTimeout = old }()

	app := fiber.New()
	Register(app, Check("slow", func(ctx context.Context) error {
		time.Sleep(3 * time.Second) // ignores ctx
		return nil
	}))

	start := time.Now()
	// Pass a 5s client timeout so app.Test doesn't cut us off before our
	// overall deadline (checkTimeout + 1s = 1.1s) fires.
	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, 5000)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	var m map[string]interface{}
	_ = json.Unmarshal(body, &m)
	code := resp.StatusCode
	elapsed := time.Since(start)

	assert.Equal(t, 503, code)
	assert.Equal(t, "error", m["status"])
	checks := m["checks"].(map[string]interface{})
	assert.Equal(t, "timeout", checks["slow"])
	assert.Less(t, elapsed, 2*time.Second) // returned well before the 3s sleep
}
