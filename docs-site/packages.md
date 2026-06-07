# Packages

Full API reference lives on
[pkg.go.dev](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers). Each
package below links to its API docs.

- [`config`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/config) — load env vars into typed structs with `default`/`required` tags.
- [`logger`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/logger) — structured logger based on zerolog.
- [`response`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/response) — standardized JSON response helpers.
- [`parser`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/parser) — Bun pagination/search query modifiers.
- [`pagination`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/pagination) — paginated result envelope with page metadata.
- [`validator`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/validator) — struct validation with custom rules and JSON field names.
- [`jwt`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/jwt) — JWT generation and validation helpers.
- [`http`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/http) — context-aware JSON HTTP client with retry.
- [`redis`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/redis) — Redis wrapper with `Remember` cache-aside plus `Delete`/`Exists`/`Expire`/`TTL`.
- [`slug`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/slug) — unique URL-safe slug generator backed by a Bun database.
- [`uploader`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/uploader) — local file uploader with size and MIME limits.
- [`middleware`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/middleware) — recover, request logging, auth, and request-id middleware.
- [`context`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/context) — request context, request-id, and type-safe local accessors.
- [`database`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/database) — Bun connector with Postgres/SQLite dialect auto-detection (plus `WithTracing`).
- [`migrate`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/migrate) — versioned migrations with `bun/migrate` and a ready cobra command.
- [`auth`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/auth) — JWT bearer auth, bcrypt, refresh tokens, scopes, and teams/roles helpers.
- [`health`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/health) — liveness (`/livez`) and readiness (`/readyz`) endpoints.
- [`metrics`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/metrics) — Prometheus request metrics middleware + `/metrics` handler.
- [`tracing`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/tracing) — OpenTelemetry tracing setup (OTLP/HTTP) + Fiber spans.
- [`jobs`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/jobs) — Redis-backed background jobs (asynq) + asynqmon monitoring mount.
- [`mailer`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/mailer) — transactional email: pluggable `Sender` (SMTP/log/memory) + template render.
- [`server`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/server) — signal-based graceful shutdown via `RunGraceful`.
- [`bootstrap`](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers/bootstrap) — one-call app wiring: middleware, health, DB, metrics, tracing, graceful shutdown.
