// Copyright 2026 Rahmad Afandi. MIT License.

package bootstrap

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/rahmadafandi/fiber-helpers/health"
	"github.com/rahmadafandi/fiber-helpers/logger"
	"github.com/rahmadafandi/fiber-helpers/middleware"
	"github.com/rahmadafandi/fiber-helpers/server"
	"github.com/uptrace/bun"
)

// App wraps *fiber.App with graceful-shutdown cleanup hooks.
type App struct {
	*fiber.App
	cleanup         []func(ctx context.Context) error
	shutdownTimeout time.Duration
	mu              sync.Mutex
	healthChecks    []health.NamedCheck
}

// Options configures the bootstrapped app. All fields are optional.
type Options struct {
	Logger          *logger.Logger
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	DB              *bun.DB
	EnableCORS      bool
	RateLimit       int
	HealthChecks    []health.NamedCheck
	FiberConfig     fiber.Config
}

// New builds a Fiber app wired with recover, request-id/context, request
// logging, optional CORS, optional rate limiting, and optional health endpoints.
func New(o Options) *App {
	if o.Logger == nil {
		o.Logger = logger.Default()
	}
	if o.RequestTimeout == 0 {
		o.RequestTimeout = 10 * time.Second
	}
	if o.ShutdownTimeout == 0 {
		o.ShutdownTimeout = 10 * time.Second
	}

	f := fiber.New(o.FiberConfig)
	f.Use(middleware.Recover(o.Logger))
	f.Use(middleware.ContextMiddleware(o.RequestTimeout))
	f.Use(middleware.RequestLogger(o.Logger))

	app := &App{App: f, shutdownTimeout: o.ShutdownTimeout}
	app.healthChecks = append(app.healthChecks, o.HealthChecks...)

	// Register health endpoints BEFORE rate limiting / CORS so liveness and
	// readiness probes are never throttled. Use a provider so checks added
	// later by Mount are included live.
	health.RegisterProvider(f, app.snapshotChecks)

	if o.EnableCORS {
		f.Use(cors.New())
	}
	if o.RateLimit > 0 {
		f.Use(limiter.New(limiter.Config{Max: o.RateLimit, Expiration: time.Minute}))
	}

	if o.DB != nil {
		db := o.DB
		app.cleanup = append(app.cleanup, func(ctx context.Context) error {
			return db.Close()
		})
	}

	return app
}

// snapshotChecks returns a copy of the current readiness checks, safe to read
// concurrently with Mount appending.
func (a *App) snapshotChecks() []health.NamedCheck {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]health.NamedCheck, len(a.healthChecks))
	copy(out, a.healthChecks)
	return out
}

// Run starts the app and blocks until shutdown, then runs cleanup hooks.
func (a *App) Run(addr string) error {
	return server.RunGraceful(a.App, addr, a.shutdownTimeout, a.cleanup...)
}
