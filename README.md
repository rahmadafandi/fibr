# Fibr

[![ci](https://github.com/rahmadafandi/fibr/actions/workflows/ci.yml/badge.svg)](https://github.com/rahmadafandi/fibr/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/rahmadafandi/fibr/branch/master/graph/badge.svg)](https://codecov.io/gh/rahmadafandi/fibr)
[![Go Reference](https://pkg.go.dev/badge/github.com/rahmadafandi/fibr.svg)](https://pkg.go.dev/github.com/rahmadafandi/fibr)
[![Go Report Card](https://goreportcard.com/badge/github.com/rahmadafandi/fibr)](https://goreportcard.com/report/github.com/rahmadafandi/fibr)
[![Release](https://img.shields.io/github/v/release/rahmadafandi/fibr)](https://github.com/rahmadafandi/fibr/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A collection of helper packages for the [Fiber](https://gofiber.io/) web framework.

📖 **[Documentation](https://rahmadafandi.github.io/fibr/)** · [API reference (pkg.go.dev)](https://pkg.go.dev/github.com/rahmadafandi/fibr)

## Install

```bash
go get github.com/rahmadafandi/fibr
```

Requires Go 1.26+. Targets Fiber v2 and Bun ORM (Postgres or SQLite).

## Stability

fibr follows [Semantic Versioning](https://semver.org/). As of **v1.0.0** the
public API is stable:

- **No breaking changes within v1.x.** Exported identifiers will not be removed
  or changed incompatibly until a v2 major release.
- New functionality arrives in backward-compatible **minor** releases; fixes in
  **patch** releases.
- Anything breaking is deferred to v2 (a new module path, `…/fibr/v2`).
- The generated-app scaffolding (`fibr new`) follows the same library version but
  generated code is a starting point you own — regenerating is never required.

Not covered by the guarantee: unexported APIs, behavior explicitly documented as
experimental, and transitive dependency internals.

## Quickstart

```go
package main

import (
    "fmt"

    "github.com/gofiber/fiber/v2"
    "github.com/rahmadafandi/fibr/bootstrap"
    "github.com/rahmadafandi/fibr/config"
    "github.com/rahmadafandi/fibr/database"
    "github.com/rahmadafandi/fibr/health"
    "github.com/rahmadafandi/fibr/response"
)

func main() {
    type Config struct {
        DatabaseURL string `mapstructure:"DATABASE_URL" default:"file::memory:?cache=shared"`
    }

    var cfg Config
    if err := config.LoadConfig(&cfg); err != nil {
        panic(err)
    }

    db, err := database.NewBun(cfg.DatabaseURL)
    if err != nil {
        panic(err)
    }

    app := bootstrap.New(bootstrap.Options{
        DB:           db,
        EnableCORS:   true,
        RateLimit:    100,
        HealthChecks: []health.NamedCheck{health.PingBun(db)},
    })

    app.Get("/", func(c *fiber.Ctx) error {
        return response.SendSuccess(c, "Hello, World!", "Welcome")
    })

    fmt.Println("Server listening on :3000")
    if err := app.Run(":3000"); err != nil {
        panic(err)
    }
}
```

## Architecture

For how the packages layer and how `bootstrap` composes them into an app, see
[ARCHITECTURE.md](ARCHITECTURE.md).

## Package Index

- [`config`](#config) — Load env vars into typed structs with `default` and `required` tags.
- [`logger`](#logger) — Structured logger based on zerolog.
- [`response`](#response) — Standardized JSON response helpers.
- [`parser`](#parser) — Bun pagination/search query modifiers, including keyset (cursor) pagination.
- [`pagination`](#pagination) — Paginated result envelope with page metadata; offset (`NewPagination`) and cursor (`CursorPage`) variants.
- [`validator`](#validator) — Struct validation with custom rules and JSON field names.
- [`bind`](#parse--validate-with-bind) — Parse and validate a request body/query/params into `T` in one call; writes `400`/`422` on failure.
- [`jwt`](#jwt) — JWT generation and validation helpers.
- [`http`](#http) — Context-aware JSON HTTP client with retry.
- [`redis`](#redis) — Redis wrapper with `Remember` cache-aside helper. Includes a `Storage` adapter (`NewStorage`) for Redis-backed rate limiting.
- [`slug`](#slug) — Unique URL-safe slug generator backed by a Bun database.
- [`uploader`](#uploader) — Local file uploader with size and MIME limits. Also includes `S3Uploader` for S3-compatible storage (AWS S3, MinIO, R2).
- [`middleware`](#middleware) — Recover, request logging, and request-id middleware.
- [`context`](#context) — Request context, request-id, and type-safe local accessors.
- [`database`](#database) — Bun connector with Postgres/SQLite dialect auto-detection.
- [`migrate`](#migrate) — Versioned migrations with `bun/migrate` and a ready cobra command.
- [`auth`](#auth) — JWT bearer authentication and bcrypt password hashing for Fiber.
- [`health`](#health) — Liveness (`/livez`) and readiness (`/readyz`) endpoints.
- [`metrics`](#metrics) — Prometheus request metrics middleware + `/metrics` handler.
- [`tracing`](#tracing) — OpenTelemetry tracing setup (OTLP/HTTP) + Fiber spans.
- [`jobs`](#jobs) — Redis-backed background jobs (asynq) + asynqmon monitoring mount. Includes `Scheduler` for cron-triggered (periodic) jobs.
- [`lock`](#lock) — Single-instance Redis distributed mutex (`TryAcquire`/`Acquire`/`Do`, owner-only `Release`/`Extend`) for single-execution across replicas.
- [`mailer`](#mailer) — Transactional email: pluggable `Sender` (SMTP/log/memory) + template render.
- [`server`](#server) — Signal-based graceful shutdown via `RunGraceful`.
- [`apierror`](#typed-errors-with-apierror) — Typed HTTP errors (`BadRequest`, `NotFound`, `Conflict`, ...) with a Fiber `ErrorHandler`; installed automatically by `bootstrap`.
- [`bootstrap`](#bootstrap) — One-call app wiring: middleware, health, DB, and graceful shutdown.

## Packages

### `config`

Loads configuration from environment variables (and a `.env` file if present) into a typed struct. Supports `default:"..."` and `required:"true"` tags, as well as `time.Duration`, float, bool, integer, and comma-separated string-slice fields.

**Usage:**

```go
import "github.com/rahmadafandi/fibr/config"

type AppConfig struct {
    Port    int           `mapstructure:"port"     default:"8080"`
    DBURL   string        `mapstructure:"db_url"   required:"true"`
    Timeout time.Duration `mapstructure:"timeout"  default:"30s"`
    Hosts   []string      `mapstructure:"hosts"    default:"a,b,c"`
}

var cfg AppConfig
err := config.LoadConfig(&cfg)
```

### `logger`

A structured logger based on [zerolog](https://github.com/rs/zerolog).

**Usage:**

```go
import "github.com/rahmadafandi/fibr/logger"

log := logger.Default()
log.Info("Hello, world!")
```

### `response`

Helper functions for sending standardized JSON responses.

**Usage:**

```go
import "github.com/rahmadafandi/fibr/response"

response.SendSuccess(c, data, "Success")
response.SendError(c, nil, "Error", 400)
```

### `parser`

Helper functions for pagination with Bun. The `Paginate` helper returns a Bun query modifier for use with `query.Apply`.

Request body/query/params parsing uses Fiber's built-in `c.BodyParser(&out)`,
`c.QueryParser(&out)`, and `c.ParamsParser(&out)` directly.

**Usage:**

```go
import "github.com/rahmadafandi/fibr/parser"

type MyStruct struct {
    Name string `json:"name"`
}

// Pagination with Bun
pq := &parser.PaginationQuery{}
if err := c.QueryParser(pq); err != nil { ... }
if err := pq.Validate([]string{"name", "created_at"}); err != nil { ... }

var rows []MyModel
err = db.NewSelect().Model(&rows).Apply(parser.Paginate(pq, []string{"name"})).Scan(ctx)

// Count with search filter
count, err := db.NewSelect().Model(&rows).Apply(parser.Count(pq.Search, []string{"name"})).Count(ctx)
```

#### Keyset (cursor) pagination

For large tables or infinite scroll, `parser.Keyset` paginates by a cursor instead of an offset — it seeks straight to the cursor position on an index, so it stays O(limit) at any depth and is stable under concurrent inserts (no shifting rows). It supports forward and backward navigation via opaque cursors.

```go
var kq parser.KeysetQuery // bound from ?limit=20&cursor=...&before=...
_ = c.QueryParser(&kq)

// The columns must form a unique total ordering; make the last a tiebreaker (the pk).
cols := []parser.KeysetColumn{{Name: "created_at", Desc: true}, {Name: "id", Desc: true}}

var rows []Article
err := db.NewSelect().Model(&rows).Apply(parser.Keyset(kq, cols)).Scan(ctx)

page := pagination.NewCursorPage(rows, kq, cols, func(a Article) []any {
    return []any{a.CreatedAt, a.ID} // values in the same order as cols
})
// page.Data, page.NextCursor (""=last), page.PrevCursor (""=first)
```

Supported cursor value types: integers, strings, bools, and `time.Time` (encoded RFC3339). Keyset trades away total/page counts — keep offset `NewPagination` when you need a count.

### `pagination`

Builds a paginated result envelope (data plus page metadata) for any element type. Guards against a zero page size and clamps page numbers below 1. For cursor-based paging, see `CursorPage` / `NewCursorPage` (above, under `parser`).

**Usage:**

```go
import "github.com/rahmadafandi/fibr/pagination"

p := pagination.NewPagination(rows, pq.Limit, pq.Page, totalCount)
// p.Data, p.PageSize, p.Count, p.TotalCount, p.PageCount, p.PageNumber, p.StartNumber
return response.SendSuccess(c, p, "ok")
```

### `validator`

A helper package for validating structs using [go-playground/validator](https://github.com/go-playground/validator). Supports custom validation rules via `Register`.

**Usage:**

```go
import "github.com/rahmadafandi/fibr/validator"

type MyStruct struct {
    Name string `json:"name" validate:"required"`
}

var body MyStruct

if errs := validator.ValidateStruct(&body); len(errs) > 0 {
    // Handle validation errors
}

// Register a custom rule (call once at startup, before concurrent use)
validator.Register("my_rule", func(fl validator.FieldLevel) bool {
    return fl.Field().String() != "forbidden"
})
```

### Parse + validate with `bind`

`bind.Body[T]`, `bind.Query[T]`, and `bind.Params[T]` decode a request into `T`
and run `validator.ValidateStruct` in one call. On malformed input they write a
`400`; on a validation failure a `422` with per-field errors; otherwise they
return `(value, true)`.

```go
type CreateInput struct {
    Email string `json:"email" validate:"required,email"`
}

func create(c *fiber.Ctx) error {
    in, ok := bind.Body[CreateInput](c)
    if !ok {
        return nil // bind wrote a 400 (malformed) or 422 (validation) response
    }
    return response.SendSuccess(c, in, "ok")
}
```

### `jwt`

A helper package for working with JSON Web Tokens.

**Usage:**

```go
import "github.com/rahmadafandi/fibr/jwt"

// Generate a token
token, err := jwt.GenerateToken(claims, secret)

// Validate a token
valid, err := jwt.ValidateToken(token, secret)
```

### `http`

A small JSON HTTP client built on [fasthttp](https://github.com/valyala/fasthttp). All request methods accept a `context.Context` and return `(statusCode int, err error)`.

**Usage:**

```go
import (
    "context"
    "time"
    fhttp "github.com/rahmadafandi/fibr/http"
)

h := fhttp.New("https://api.example.com",
    fhttp.WithTimeout(10*time.Second),
    fhttp.WithRetry(3, 500*time.Millisecond),
    fhttp.WithHeader("Authorization", "Bearer "+token),
)

var result MyResponse
code, err := h.Get(ctx, "/resource/1", &result)

code, err = h.Post(ctx, "/resource", requestBody, &result)

// Fire and forget (background, non-blocking)
h.FireAndForget(ctx, fhttp.Post, "/events", eventPayload)
```

### `redis`

A Redis wrapper with JSON serialization helpers and a generic `Remember` cache-aside function.

**Usage:**

```go
import (
    "context"
    "time"
    "github.com/redis/go-redis/v9"
    firedis "github.com/rahmadafandi/fibr/redis"
)

opt, err := firedis.ParseRedisOptions("redis://localhost:6379/0")
if err != nil {
    // handle invalid URL
}
rds := firedis.New(redis.NewClient(opt))

// Set / Get
_ = rds.Set(ctx, "key", myValue, time.Minute)
_ = rds.Get(ctx, "key", &myValue)

// Cache-aside: returns cached value or calls loader on miss
result, err := firedis.Remember(ctx, rds, "key", time.Minute, func() (MyType, error) {
    return expensiveLoad()
})
```

Invalidate and inspect entries with `Delete(ctx, keys...)`, `Exists(ctx, key)`,
`Expire(ctx, key, ttl)`, and `TTL(ctx, key)`.

### Redis-backed rate limiting

```go
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
app := bootstrap.New(bootstrap.Options{
    RateLimit:        100,
    RateLimitStorage: fibrredis.NewStorage(client), // shared across instances
})
```

### `slug`

Generates a unique, URL-safe slug for a given table using a [Bun](https://bun.uptrace.dev/) database.

**Usage:**

```go
import (
    "context"
    "github.com/rahmadafandi/fibr/slug"
)

// Returns e.g. "my-first-post-abc123defgh456"
s, err := slug.Generate(ctx, db, "posts", "My First Post")
```

### `uploader`

A helper package for uploading files to local storage. `NewLocalUploader` accepts functional options for max file size and MIME type allowlist. The filename is sanitized automatically.

**Usage:**

```go
import "github.com/rahmadafandi/fibr/uploader"

// Create a local uploader (max 5 MB, images only)
up := uploader.NewLocalUploader("./uploads",
    uploader.WithMaxSize(5<<20),
    uploader.WithAllowedMime([]string{"image/jpeg", "image/png"}),
)

// Upload a file (filename is sanitized before saving)
path, err := up.Upload(file, filename)
```

### S3-compatible uploads with `uploader.S3Uploader`

```go
client, _ := minio.New(endpoint, &minio.Options{Creds: creds, Secure: true})
up := uploader.NewS3Uploader(client, "my-bucket",
    uploader.WithKeyPrefix("avatars/"),
    uploader.WithBaseURL("https://cdn.example.com/"),
)
url, err := up.Upload(file, "photo.png") // -> https://cdn.example.com/avatars/photo.png
```

### `middleware`

A collection of useful middleware.

**Usage:**

```go
import "github.com/rahmadafandi/fibr/middleware"

app := fiber.New()

// Recover from panics
app.Use(middleware.Recover(logger))

// Log requests
app.Use(middleware.RequestLogger(logger))

// Protect routes (JWT bearer + revocation/scopes)
app.Use(auth.RequireAuth(secret))

// Context
app.Use(middleware.ContextMiddleware(10 * time.Second))
```

### `context`

Accessors for values stored on the Fiber context: the request-scoped
`context.Context`, the request ID, and type-safe locals.

**Usage:**

```go
import fhctx "github.com/rahmadafandi/fibr/context"

ctx := fhctx.GetContext(c)        // request-scoped context.Context
id := fhctx.GetRequestID(c)       // request id (set by ContextMiddleware)

fhctx.SetLocal(c, "user", user)   // store a typed value
u := fhctx.GetLocal[User](c, "user") // retrieve it (zero value if absent)
```

### `database`

Opens a [Bun](https://bun.uptrace.dev/) database, picking the dialect from the DSN
(`postgres://` → Postgres, `file:`/`:memory:`/path → SQLite).

```go
import "github.com/rahmadafandi/fibr/database"

db, err := database.NewBun("postgres://localhost/app",
    database.WithMaxOpenConns(20),
    database.WithPingTimeout(3*time.Second),
)
```

### `migrate`

Versioned database migrations over `bun/migrate`. Declare a collection, register
Go migrations, and run them from your app binary.

```go
// internal/migrations/migrations.go
var Migrations = migrate.NewMigrations(migrate.WithMigrationsDirectory("internal/migrations"))

// cmd/api/main.go
root.AddCommand(migrate.NewCommand(openDB, migrations.Migrations))
```

`migrate.NewCommand` gives `up`, `down`, `status`, and `create <name>`
subcommands. The core funcs `Up`/`Down`/`Status`/`Create` are also usable
directly. Each migration file registers itself in `init()` via
`Migrations.MustRegister(up, down)`; the version comes from the filename
(`<timestamp>_<name>.go`).

### `auth`

JWT bearer authentication and bcrypt password hashing for Fiber.

```go
hash, _ := auth.Hash(password)          // bcrypt
err := auth.Compare(hash, password)     // nil = match

app.Get("/me", auth.RequireAuth(secret), handler)            // 401 if no/invalid token
app.Get("/admin", auth.RequireAuth(secret),
    auth.RequireScope("admin"), handler)                     // 403 if scope missing

claims, ok := auth.Claims(c)            // jwt.MapClaims stored by the middleware
sub := auth.Subject(c)                  // claims["sub"]
scopes := auth.Scopes(c)                // normalized []string
```

`Optional(secret)` validates a token when present but never rejects. Scopes are a
`scopes` JWT claim (`[]string`). Tokens are minted with the `jwt` package
(`jwt.GenerateTokenWithExpiry`).

### `health`

Liveness (`/livez`) and readiness (`/readyz`) endpoints with concurrent checks.

```go
import "github.com/rahmadafandi/fibr/health"

health.Register(app, health.PingBun(db),
    health.Check("cache", func(ctx context.Context) error { return rds.Ping(ctx) }),
)
// GET /livez  -> 200 {"status":"ok"}
// GET /readyz -> 200/503 {"status":"...","checks":{...}}
```

### `metrics`

Prometheus request metrics. Standalone:

```go
import "github.com/rahmadafandi/fibr/metrics"

app.Use(metrics.Middleware())
app.Get("/metrics", metrics.Handler())
```

Records `http_requests_total{method,path,status}` and
`http_request_duration_seconds{...}`. The `path` label is the Fiber route
template (e.g. `/items/:id`), so cardinality stays bounded. The default registry
also exposes Go-runtime and process collectors (`go_goroutines`, GC, memory,
fds). The middleware skips its own `/metrics` path.

Via `bootstrap`, enable with `Options{Metrics: true}` — it installs the
middleware and registers `/metrics` ahead of the rate limiter. In a generated
app, set `METRICS_ENABLED=true`.

### tracing

OpenTelemetry distributed tracing. Set up the provider once at startup and defer
its shutdown:

```go
import "github.com/rahmadafandi/fibr/tracing"

shutdown, err := tracing.Setup(ctx, tracing.WithServiceName("my-svc"))
if err != nil { /* handle */ }
defer shutdown(context.Background())
```

`Setup` builds an OTLP/HTTP exporter (configured by the standard `OTEL_` env vars
like `OTEL_EXPORTER_OTLP_ENDPOINT` / `OTEL_SERVICE_NAME`) and installs the global
tracer provider + W3C propagator. Via `bootstrap`, enable with
`Options{Tracing: true}` (installs the `otelfiber` server-span middleware) and
pass `shutdown` through `Options{Cleanup: []func(context.Context) error{shutdown}}`
for graceful shutdown. When tracing is active, `RequestLogger` adds `trace_id` /
`span_id` to request logs. In a generated app, set `TRACING_ENABLED=true`.

When tracing is enabled, generated apps also install Bun's `bunotel` query hook
(`database.WithTracing()`), so each SQL query becomes a span nested under the
request span.

### jobs

Redis-backed background jobs built on [asynq](https://github.com/hibiken/asynq).

```go
import "github.com/rahmadafandi/fibr/jobs"

opt, _ := jobs.RedisConnOpt(os.Getenv("REDIS_URL"))

// enqueue (HTTP side)
client := jobs.NewClient(opt)
client.Enqueue(ctx, "welcome:send", WelcomePayload{Email: "a@b.com"})

// process (worker side)
srv := jobs.NewServer(opt, jobs.ServerConfig{Concurrency: 10})
jobs.Handle[WelcomePayload](srv, "welcome:send", handleWelcome)
srv.Run()
```

`Enqueue` JSON-marshals the payload; the generic `Handle[T]` decodes it back into
`T` before calling your handler (a malformed payload wraps `asynq.SkipRetry` so it
is not retried forever). Mount the
[asynqmon](https://github.com/hibiken/asynqmon) dashboard through `bootstrap`
(the asynqmon dependency stays out of `bootstrap` itself):

```go
bootstrap.New(bootstrap.Options{
    Asynqmon: &bootstrap.AsynqmonMount{
        Handler: jobs.MonitoringHandler(opt, "/monitoring"),
        Path:    "/monitoring",
    },
})
```

Generate an app with the queue scaffolded via `fibr new --queue`: it adds
a `worker` subcommand, a sample job, the monitoring UI mount, and the
`REDIS_URL` / `QUEUE_CONCURRENCY` / `ASYNQMON_PATH` config keys. When `REDIS_URL`
is unset the queue is disabled with a startup warning (the `worker` subcommand
exits with an error).

### Scheduled jobs with `jobs.Scheduler`

```go
sched := jobs.NewScheduler(opt) // opt from jobs.RedisConnOpt(redisURL)
if _, err := sched.Register("0 2 * * *", "cleanup:run", CleanupPayload{OlderThanDays: 30}); err != nil {
    log.Fatal(err)
}
log.Fatal(sched.Run()) // run ONE instance; workers process the enqueued tasks
```

### `lock`

A single-instance Redis distributed mutex. Across multiple replicas, it guarantees that a unit of work runs on at most one of them at a time — for example, so a `jobs.Scheduler` cron task does not fire once per replica.

```go
locker := lock.New(redisClient) // redisClient is a redis.UniversalClient

// Run-once across replicas: acquires, runs fn, releases. Returns
// lock.ErrNotAcquired (without running fn) if another replica holds it.
err := locker.Do(ctx, "cron:nightly-cleanup", 30*time.Second, func() error {
    return cleanup(ctx)
})
if err != nil && !errors.Is(err, lock.ErrNotAcquired) {
    log.Fatal(err)
}
```

For finer control: `TryAcquire` (one non-blocking attempt), `Acquire` (blocks until the lock is free or the context ends), and on the returned handle `Extend` (renew the TTL for long-running work) and `Release`. Release and Extend are owner-only — a token guards against deleting or renewing a lock another replica has since taken over.

### mailer

Transactional email through a pluggable `Sender`.

```go
import "github.com/rahmadafandi/fibr/mailer"

sender, _ := mailer.New(mailer.SMTPConfig{
    Host: os.Getenv("SMTP_HOST"), Port: 587,
    Username: os.Getenv("SMTP_USERNAME"), Password: os.Getenv("SMTP_PASSWORD"),
    From: "no-reply@example.com",
})

html, text, _ := mailer.Render("<p>Hi {{.Name}}</p>", "Hi {{.Name}}", data)
sender.Send(ctx, mailer.Message{To: []string{"a@b.com"}, Subject: "Hi", HTML: html, Text: text})
```

`New` returns an SMTP sender when `Host` is set, otherwise a `LogSender` that
logs instead of sending (handy in development). A `MemorySender` captures
messages for tests. Because `Message` is JSON-serializable it doubles as an
asynq job payload — generated apps with both `--mailer` and `--queue` send
asynchronously through an `email:send` job; with `--mailer` alone they send
inline. `fibr new --mailer` also sends the team invitation email (with
`--auth-with-team`) and the welcome job (with `--queue`).

### `server`

Signal-based graceful shutdown.

```go
import "github.com/rahmadafandi/fibr/server"

err := server.RunGraceful(app, ":3000", 10*time.Second, func(ctx context.Context) error {
    return db.Close()
})
```

### Typed errors with `apierror`

`apierror.NotFound("...")`, `Conflict`, `Unauthorized`, ... return typed `*Error` values. `bootstrap` installs `apierror.Handler` as the default `ErrorHandler`, so returning one from a handler renders a consistent JSON error.

```go
func getUser(c *fiber.Ctx) error {
    u, err := svc.Find(c.UserContext(), id)
    if err != nil {
        return apierror.NotFound("user not found").WithCode("user_not_found")
    }
    return response.SendSuccess(c, u, "ok")
}
// bootstrap.New installs apierror.Handler, so the return renders as:
// {"code":404,"message":"user not found","error":"user_not_found","status":"error"}
```

### `bootstrap`

Optional one-call wiring of recover, request id, logging, optional CORS / rate
limit / health, and graceful shutdown.

```go
import "github.com/rahmadafandi/fibr/bootstrap"

app := bootstrap.New(bootstrap.Options{
    DB:           db,
    EnableCORS:   true,
    RateLimit:    100,
    HealthChecks: []health.NamedCheck{health.PingBun(db)},
})
app.Get("/", handler)
log.Fatal(app.Run(":3000")) // graceful shutdown + db.Close handled
```

### Modules

A `Module` is a self-contained feature that registers its own routes. Mount it
in one line; it can optionally migrate its tables and report health.

```go
type Module interface {
    Name() string
    Register(r fiber.Router) error
}
// optional, detected via type assertion:
type Migrator      interface{ Migrate(ctx context.Context) error }
type HealthChecker interface{ Checks() []health.NamedCheck }

app := bootstrap.New(bootstrap.Options{DB: db})
if err := app.Mount(user.NewUserModule(db), product.NewProductModule(db)); err != nil {
    log.Fatal(err)
}
```

By default `Mount` does **not** create tables — use migrations
(`migrate.NewCommand`). Set `bootstrap.Options{AutoMigrate: true}` (e.g. from an
`AUTO_MIGRATE` env in dev) to have `Mount` run each module's `Migrate` at startup.

`Mount` registers each module's routes and, when `AutoMigrate` is enabled, runs
its `Migrate`. It also adds its `Checks()` to `/readyz`.

> Note: `*bootstrap.App.Mount` is the module-aware method and shadows Fiber's
> `Mount`. Where a `fiber.Router` is needed (e.g. passing the app to a route
> registrar), use the embedded `app.App`.

## License

[MIT](LICENSE) © 2026 Rahmad Afandi
