# Packages

Full API reference lives on
[pkg.go.dev](https://pkg.go.dev/github.com/rahmadafandi/fibr). Each
package below links to its API docs.

- [`config`](https://pkg.go.dev/github.com/rahmadafandi/fibr/config) — load env vars into typed structs with `default`/`required` tags.
- [`logger`](https://pkg.go.dev/github.com/rahmadafandi/fibr/logger) — structured logger based on zerolog.
- [`response`](https://pkg.go.dev/github.com/rahmadafandi/fibr/response) — standardized JSON response helpers.
- [`parser`](https://pkg.go.dev/github.com/rahmadafandi/fibr/parser) — Bun pagination/search query modifiers.
- [`pagination`](https://pkg.go.dev/github.com/rahmadafandi/fibr/pagination) — paginated result envelope with page metadata.
- [`validator`](https://pkg.go.dev/github.com/rahmadafandi/fibr/validator) — struct validation with custom rules and JSON field names.
- [`bind`](https://pkg.go.dev/github.com/rahmadafandi/fibr/bind) — parse and validate a request body/query/params into `T` in one call; writes `400`/`422` on failure.
- [`jwt`](https://pkg.go.dev/github.com/rahmadafandi/fibr/jwt) — JWT generation and validation helpers.
- [`http`](https://pkg.go.dev/github.com/rahmadafandi/fibr/http) — context-aware JSON HTTP client with retry.
- [`redis`](https://pkg.go.dev/github.com/rahmadafandi/fibr/redis) — Redis wrapper with `Remember` cache-aside plus `Delete`/`Exists`/`Expire`/`TTL`. Includes a `Storage` adapter (fiber.Storage) for Redis-backed rate limiting.

  `redis.NewStorage(client)` adapts a go-redis client to `fiber.Storage` — pass it as `bootstrap.Options.RateLimitStorage` for a rate limiter consistent across instances.
- [`slug`](https://pkg.go.dev/github.com/rahmadafandi/fibr/slug) — unique URL-safe slug generator backed by a Bun database.
- [`uploader`](https://pkg.go.dev/github.com/rahmadafandi/fibr/uploader) — local file uploader with size and MIME limits. Includes `S3Uploader` for S3-compatible storage (AWS S3, MinIO, R2).
- [`middleware`](https://pkg.go.dev/github.com/rahmadafandi/fibr/middleware) — recover, request logging, auth, and request-id middleware.
- [`context`](https://pkg.go.dev/github.com/rahmadafandi/fibr/context) — request context, request-id, and type-safe local accessors.
- [`database`](https://pkg.go.dev/github.com/rahmadafandi/fibr/database) — Bun connector with Postgres/SQLite dialect auto-detection (plus `WithTracing`).
- [`migrate`](https://pkg.go.dev/github.com/rahmadafandi/fibr/migrate) — versioned migrations with `bun/migrate` and a ready cobra command.
- [`auth`](https://pkg.go.dev/github.com/rahmadafandi/fibr/auth) — JWT bearer auth, bcrypt, refresh tokens, scopes, and teams/roles helpers.
- [`health`](https://pkg.go.dev/github.com/rahmadafandi/fibr/health) — liveness (`/livez`) and readiness (`/readyz`) endpoints.
- [`metrics`](https://pkg.go.dev/github.com/rahmadafandi/fibr/metrics) — Prometheus request metrics middleware + `/metrics` handler.
- [`tracing`](https://pkg.go.dev/github.com/rahmadafandi/fibr/tracing) — OpenTelemetry tracing setup (OTLP/HTTP) + Fiber spans.
- [`jobs`](https://pkg.go.dev/github.com/rahmadafandi/fibr/jobs) — Redis-backed background jobs (asynq) + asynqmon monitoring mount. Includes `Scheduler` for cron-triggered (periodic) jobs.
- [`mailer`](https://pkg.go.dev/github.com/rahmadafandi/fibr/mailer) — transactional email: pluggable `Sender` (SMTP/log/memory) + template render.
- [`server`](https://pkg.go.dev/github.com/rahmadafandi/fibr/server) — signal-based graceful shutdown via `RunGraceful`.
- [`apierror`](https://pkg.go.dev/github.com/rahmadafandi/fibr/apierror) — typed HTTP errors (`NotFound`, `Conflict`, ...) + a Fiber `ErrorHandler` that renders them as the JSON envelope; wired by `bootstrap` automatically.
- [`bootstrap`](https://pkg.go.dev/github.com/rahmadafandi/fibr/bootstrap) — one-call app wiring: middleware, health, DB, metrics, tracing, security headers (helmet), compression, idempotency keys, graceful shutdown.
