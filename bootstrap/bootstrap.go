// Copyright 2026 Rahmad Afandi. MIT License.

package bootstrap

import (
	"context"
	"net/http"
	"sync"
	"time"

	otelfiber "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/rahmadafandi/fibr/apierror"
	"github.com/rahmadafandi/fibr/health"
	"github.com/rahmadafandi/fibr/logger"
	"github.com/rahmadafandi/fibr/metrics"
	"github.com/rahmadafandi/fibr/middleware"
	"github.com/rahmadafandi/fibr/openapi"
	"github.com/rahmadafandi/fibr/server"
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
	Logger             *logger.Logger
	RequestTimeout     time.Duration
	ShutdownTimeout    time.Duration
	DB                 *bun.DB
	EnableCORS         bool
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
	f.Use(middleware.ContextMiddleware(o.RequestTimeout))
	if o.SecurityHeaders {
		f.Use(helmet.New())
	}
	if o.Compression {
		f.Use(compress.New())
	}
	if o.Idempotency {
		f.Use(idempotency.New(idempotency.Config{Storage: o.IdempotencyStorage}))
	}
	if o.Tracing {
		f.Use(otelfiber.Middleware())
	}
	f.Use(middleware.RequestLogger(o.Logger))
	if o.Metrics {
		f.Use(metrics.Middleware())
	}

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
		args := make([]interface{}, 0, len(handlers)+1)
		args = append(args, path)
		for _, h := range handlers {
			args = append(args, h)
		}
		f.Use(args...)
	}

	if o.EnableCORS {
		f.Use(cors.New())
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
