// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package health

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

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
