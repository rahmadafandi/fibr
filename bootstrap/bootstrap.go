// Copyright 2026 Rahmad Afandi. MIT License.

// Package bootstrap wires a Fiber app — middleware, health, metrics, tracing, OpenAPI docs, i18n, and graceful shutdown — from a single Options struct.
package bootstrap

import (
	"context"
	"net/http"
	"sync"
	"time"

	fiberotel "github.com/gofiber/contrib/v3/otel"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/idempotency"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/rahmadafandi/fibr/v2/apierror"
	"github.com/rahmadafandi/fibr/v2/health"
	"github.com/rahmadafandi/fibr/v2/i18n"
	"github.com/rahmadafandi/fibr/v2/logger"
	"github.com/rahmadafandi/fibr/v2/metrics"
	"github.com/rahmadafandi/fibr/v2/middleware"
	"github.com/rahmadafandi/fibr/v2/openapi"
	"github.com/rahmadafandi/fibr/v2/server"
	"github.com/uptrace/bun"
)

// App wraps *fiber.App with graceful-shutdown cleanup hooks.
type App struct {
	*fiber.App
	cleanup         []func(ctx context.Context) error
	shutdownTimeout time.Duration
	mu              sync.Mutex
	healthChecks    []health.NamedCheck
	autoMigrate     bool
}

// AsynqmonMount configures mounting an external monitoring UI handler (e.g. the
// asynqmon dashboard) on the app. Kept generic so bootstrap need not import
// asynqmon.
type AsynqmonMount struct {
	Handler    http.Handler    // the UI handler, e.g. jobs.MonitoringHandler(opt, path)
	Path       string          // mount path; defaults to "/monitoring"
	Middleware []fiber.Handler // optional guards applied before the handler
}

// Options configures the bootstrapped app. All fields are optional.
type Options struct {
	Logger          *logger.Logger
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	DB              *bun.DB
	// CORSOrigins, if non-empty, enables CORS for exactly these origins
	// (e.g. "https://app.example.com"). CORS is off when both this and CORS
	// are unset.
	CORSOrigins []string
	// CORS, if non-nil, enables CORS with this full configuration and takes
	// precedence over CORSOrigins.
	CORS               *cors.Config
	RateLimit          int
	RateLimitStorage   fiber.Storage
	SecurityHeaders    bool
	Compression        bool
	Idempotency        bool
	IdempotencyStorage fiber.Storage
	AutoMigrate        bool
	Metrics            bool
	Tracing            bool
	Cleanup            []func(context.Context) error
	Asynqmon           *AsynqmonMount
	HealthChecks       []health.NamedCheck
	FiberConfig        fiber.Config

	// OpenAPI, if non-nil, mounts the OpenAPI document at OpenAPISpecURL and a
	// Swagger UI at OpenAPIDocsURL.
	OpenAPI        *openapi.Spec
	OpenAPISpecURL string // default "/openapi.json"
	OpenAPIDocsURL string // default "/docs"

	// I18n, if set, mounts the locale-detection middleware so handlers can use
	// i18n.T / i18n.Locale.
	I18n *i18n.Bundle
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

	if o.FiberConfig.ErrorHandler == nil {
		o.FiberConfig.ErrorHandler = apierror.Handler
	}
	f := fiber.New(o.FiberConfig)
	f.Use(middleware.Recover(o.Logger))
	// Tracing goes before ContextMiddleware so the request context it builds
	// is a child of the request span.
	if o.Tracing {
		f.Use(fiberotel.New())
	}
	f.Use(middleware.ContextMiddleware(o.RequestTimeout))
	if o.I18n != nil {
		f.Use(i18n.Middleware(o.I18n))
	}
	if o.SecurityHeaders {
		f.Use(helmet.New())
	}
	if o.Compression {
		f.Use(compress.New())
	}
	if o.Idempotency {
		f.Use(idempotency.New(idempotency.Config{Storage: o.IdempotencyStorage}))
	}
	f.Use(middleware.RequestLogger(o.Logger))
	if o.Metrics {
		f.Use(metrics.Middleware())
	}
	// A second Recover, inside the logger and metrics, turns a handler panic
	// into a 500 response before it unwinds through them, so the request is
	// still logged and counted. The outer one stays as the safety net for
	// panics in the middleware above.
	f.Use(middleware.Recover(o.Logger))

	app := &App{App: f, shutdownTimeout: o.ShutdownTimeout, autoMigrate: o.AutoMigrate}
	app.healthChecks = append(app.healthChecks, o.HealthChecks...)

	// Register health endpoints BEFORE rate limiting / CORS so liveness and
	// readiness probes are never throttled. Use a provider so checks added
	// later by Mount are included live.
	health.RegisterProvider(f, app.snapshotChecks)

	if o.Metrics {
		f.Get(metrics.MetricsPath, metrics.Handler())
	}

	if o.Asynqmon != nil && o.Asynqmon.Handler != nil {
		path := o.Asynqmon.Path
		if path == "" {
			path = "/monitoring"
		}
		handlers := append(append([]fiber.Handler{}, o.Asynqmon.Middleware...),
			adaptor.HTTPHandler(o.Asynqmon.Handler))
		args := make([]any, 0, len(handlers)+1)
		args = append(args, path)
		for _, h := range handlers {
			args = append(args, h)
		}
		f.Use(args...)
	}

	if o.CORS != nil {
		f.Use(cors.New(*o.CORS))
	} else if len(o.CORSOrigins) > 0 {
		f.Use(cors.New(cors.Config{AllowOrigins: o.CORSOrigins}))
	}
	if o.RateLimit > 0 {
		f.Use(limiter.New(limiter.Config{Max: o.RateLimit, Expiration: time.Minute, Storage: o.RateLimitStorage}))
	}

	if o.DB != nil {
		db := o.DB
		app.cleanup = append(app.cleanup, func(ctx context.Context) error {
			return db.Close()
		})
	}
	app.cleanup = append(app.cleanup, o.Cleanup...)

	if o.OpenAPI != nil {
		specURL := o.OpenAPISpecURL
		if specURL == "" {
			specURL = "/openapi.json"
		}
		docsURL := o.OpenAPIDocsURL
		if docsURL == "" {
			docsURL = "/docs"
		}
		f.Get(specURL, o.OpenAPI.SpecHandler())
		f.Get(docsURL, o.OpenAPI.UIHandler(specURL))
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
