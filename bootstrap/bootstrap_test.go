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

package bootstrap

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/database"
	"github.com/rahmadafandi/fiber-helpers/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRoutesWork(t *testing.T) {
	app := New(Options{})
	app.Get("/ping", func(c *fiber.Ctx) error { return c.SendString("pong") })

	resp, err := app.Test(httptest.NewRequest("GET", "/ping", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestNewWiresHealth(t *testing.T) {
	app := New(Options{
		HealthChecks: []health.NamedCheck{
			health.Check("ok", func(ctx context.Context) error { return nil }),
		},
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/livez", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = app.Test(httptest.NewRequest("GET", "/readyz", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestNewSetsRequestIDHeader(t *testing.T) {
	app := New(Options{})
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

func TestNewDBAddsCleanup(t *testing.T) {
	db, err := database.NewBun("file::memory:?cache=shared")
	require.NoError(t, err)
	app := New(Options{DB: db})
	assert.Len(t, app.cleanup, 1)
}

func TestNewNoDBNoCleanup(t *testing.T) {
	app := New(Options{})
	assert.Empty(t, app.cleanup)
}
