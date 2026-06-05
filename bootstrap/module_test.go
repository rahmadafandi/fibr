// Copyright 2026 Rahmad Afandi. MIT License.

package bootstrap

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeModule implements Module, Migrator, and HealthChecker.
type fakeModule struct {
	name        string
	migrated    bool
	registered  bool
	migrateErr  error
	registerErr error
	checkErr    error
}

func (m *fakeModule) Name() string { return m.name }
func (m *fakeModule) Register(r fiber.Router) error {
	m.registered = true
	r.Get("/"+m.name, func(c *fiber.Ctx) error { return c.SendString(m.name) })
	return m.registerErr
}
func (m *fakeModule) Migrate(ctx context.Context) error {
	m.migrated = true
	return m.migrateErr
}
func (m *fakeModule) Checks() []health.NamedCheck {
	return []health.NamedCheck{health.Check(m.name, func(ctx context.Context) error { return m.checkErr })}
}

func TestMountRegistersAndMigrates(t *testing.T) {
	app := New(Options{})
	m := &fakeModule{name: "widget"}
	require.NoError(t, app.Mount(m))
	assert.True(t, m.migrated, "Migrate should be called")
	assert.True(t, m.registered, "Register should be called")

	resp, err := app.Test(httptest.NewRequest("GET", "/widget", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestMountChecksReachReadyz(t *testing.T) {
	app := New(Options{})
	require.NoError(t, app.Mount(&fakeModule{name: "gadget", checkErr: assert.AnError}))

	resp, err := app.Test(httptest.NewRequest("GET", "/readyz", nil))
	require.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode, "failing module check should fail readiness")
}

func TestMountMigrateErrorWrapped(t *testing.T) {
	app := New(Options{})
	err := app.Mount(&fakeModule{name: "broke", migrateErr: assert.AnError})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broke")
	assert.ErrorIs(t, err, assert.AnError)
}

func TestMountRegisterErrorWrapped(t *testing.T) {
	app := New(Options{})
	err := app.Mount(&fakeModule{name: "rbroke", registerErr: assert.AnError})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rbroke")
	assert.ErrorIs(t, err, assert.AnError)
}

// Module with no optional capabilities must still mount.
type bareModule struct{}

func (bareModule) Name() string                  { return "bare" }
func (bareModule) Register(r fiber.Router) error { return nil }

func TestMountBareModule(t *testing.T) {
	app := New(Options{})
	require.NoError(t, app.Mount(bareModule{}))
}
