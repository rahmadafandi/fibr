// Copyright 2026 Rahmad Afandi. MIT License.

package bootstrap

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/health"
)

// Module is a self-contained feature that registers its own routes. Implement
// the optional Migrator and HealthChecker interfaces to opt into startup
// migration and readiness reporting.
type Module interface {
	// Name identifies the module; it is used in error messages from Mount.
	Name() string
	// Register mounts the module's routes on r.
	Register(r fiber.Router) error
}

// Migrator is an optional Module capability: create/migrate tables on startup.
type Migrator interface {
	Migrate(ctx context.Context) error
}

// HealthChecker is an optional Module capability: contribute readiness checks.
type HealthChecker interface {
	Checks() []health.NamedCheck
}

// Mount wires each module into the app. For every module it, in order:
//  1. runs Migrate(ctx) if the module implements Migrator AND the app was
//     built with Options{AutoMigrate: true}; when AutoMigrate is false the
//     schema is expected to be managed by external migration tooling,
//  2. calls Register to mount routes,
//  3. collects Checks() if the module implements HealthChecker (they then
//     appear in /readyz).
//
// It stops and returns a wrapped error on the first failure.
func (a *App) Mount(mods ...Module) error {
	ctx := context.Background()
	for _, m := range mods {
		if a.autoMigrate {
			if mg, ok := m.(Migrator); ok {
				if err := mg.Migrate(ctx); err != nil {
					return fmt.Errorf("mount %s: migrate: %w", m.Name(), err)
				}
			}
		}
		if err := m.Register(a.App); err != nil {
			return fmt.Errorf("mount %s: register: %w", m.Name(), err)
		}
		if hc, ok := m.(HealthChecker); ok {
			a.mu.Lock()
			a.healthChecks = append(a.healthChecks, hc.Checks()...)
			a.mu.Unlock()
		}
	}
	return nil
}
