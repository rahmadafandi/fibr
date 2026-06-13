# Packages

Full API reference lives on
[pkg.go.dev](https://pkg.go.dev/github.com/rahmadafandi/fibr). Each
package below links to its API docs.

- [`config`](https://pkg.go.dev/github.com/rahmadafandi/fibr/config) — load env vars into typed structs with `default`/`required` tags.
- [`logger`](https://pkg.go.dev/github.com/rahmadafandi/fibr/logger) — structured logger based on zerolog.
- [`response`](https://pkg.go.dev/github.com/rahmadafandi/fibr/response) — standardized JSON response helpers.
- [`parser`](https://pkg.go.dev/github.com/rahmadafandi/fibr/parser) — Bun pagination/search query modifiers, including keyset (cursor) pagination (`Keyset`, `EncodeCursor`/`DecodeCursor`).
- [`pagination`](https://pkg.go.dev/github.com/rahmadafandi/fibr/pagination) — paginated result envelope with page metadata: offset (`NewPagination`) and cursor (`CursorPage`/`NewCursorPage`, forward + backward) variants.
- [`validator`](https://pkg.go.dev/github.com/rahmadafandi/fibr/validator) — struct validation with custom rules and JSON field names.
- [`bind`](https://pkg.go.dev/github.com/rahmadafandi/fibr/bind) — parse and validate a request body/query/params into `T` in one call; writes `400`/`422` on failure.
- [`jwt`](https://pkg.go.dev/github.com/rahmadafandi/fibr/jwt) — JWT generation and validation helpers.
- [`http`](https://pkg.go.dev/github.com/rahmadafandi/fibr/http) — context-aware JSON HTTP client with retry and an optional circuit breaker (`WithCircuitBreaker`), plus `PostForm` and `PostMultipart` (file upload).
- [`redis`](https://pkg.go.dev/github.com/rahmadafandi/fibr/redis) — Redis wrapper with `Remember` cache-aside (singleflight-deduped against cache stampede) plus `Delete`/`Exists`/`Expire`/`TTL` and bulk/atomic ops (`MSet`/`MGet`/`Incr`/`Decr`/`SetNX`/`GetSet`). Includes a `Storage` adapter (fiber.Storage) for Redis-backed rate limiting, and `Publish`/`Subscribe[T]` pub/sub for cross-instance events.

  `redis.NewStorage(client)` adapts a go-redis client to `fiber.Storage` — pass it as `bootstrap.Options.RateLimitStorage` for a rate limiter consistent across instances.
- [`lock`](https://pkg.go.dev/github.com/rahmadafandi/fibr/lock) — single-instance Redis distributed mutex: `TryAcquire`/`Acquire`/`Do` (run-once across replicas) with owner-only `Release`/`Extend`. Guards single-execution of scheduler/cron work in multi-replica deploys.
- [`outbox`](https://pkg.go.dev/github.com/rahmadafandi/fibr/outbox) — transactional outbox: `Enqueue` an event in the same Bun transaction as your business write, and a background `Relay` publishes pending events at-least-once (`NewRedisPublisher`, optional single-relay coordination via `lock`). Solves the dual-write problem.
- [`events`](https://pkg.go.dev/github.com/rahmadafandi/fibr/events) — in-process typed event bus: `Subscribe[T]` / `Publish[T]` over a `Bus`, synchronous by default (joined errors) with an opt-in async mode (`WithAsync`). Complements `outbox` for in-memory, intra-process fan-out.
- [`slug`](https://pkg.go.dev/github.com/rahmadafandi/fibr/slug) — unique URL-safe slug generator backed by a Bun database.
- [`uploader`](https://pkg.go.dev/github.com/rahmadafandi/fibr/uploader) — local file uploader with size and MIME limits. Includes `S3Uploader` for S3-compatible storage (AWS S3, MinIO, R2).
- [`middleware`](https://pkg.go.dev/github.com/rahmadafandi/fibr/middleware) — recover, request logging, and request-id middleware.
- [`context`](https://pkg.go.dev/github.com/rahmadafandi/fibr/context) — request context, request-id, and type-safe local accessors.
- [`database`](https://pkg.go.dev/github.com/rahmadafandi/fibr/database) — Bun connector with Postgres/SQLite dialect auto-detection (plus `WithTracing`).
- [`migrate`](https://pkg.go.dev/github.com/rahmadafandi/fibr/migrate) — versioned migrations with `bun/migrate` and a ready cobra command.
- [`auth`](https://pkg.go.dev/github.com/rahmadafandi/fibr/auth) — JWT bearer auth, bcrypt, refresh tokens, scopes, and teams/roles helpers.
- [`health`](https://pkg.go.dev/github.com/rahmadafandi/fibr/health) — liveness (`/livez`) and readiness (`/readyz`) endpoints, dependency probes (`PingBun`/`PingRedis`/`PingHTTP`/`PingTCP`), and a `ReadinessGate` for drain-on-shutdown.
- [`metrics`](https://pkg.go.dev/github.com/rahmadafandi/fibr/metrics) — Prometheus request metrics middleware + `/metrics` handler.
- [`tracing`](https://pkg.go.dev/github.com/rahmadafandi/fibr/tracing) — OpenTelemetry tracing setup (OTLP/HTTP) + Fiber spans.
- [`jobs`](https://pkg.go.dev/github.com/rahmadafandi/fibr/jobs) — Redis-backed background jobs (asynq) + asynqmon monitoring mount. Includes `Scheduler` for cron-triggered (periodic) jobs.
- [`mailer`](https://pkg.go.dev/github.com/rahmadafandi/fibr/mailer) — transactional email: pluggable `Sender` (SMTP/log/memory) + template render.
- [`server`](https://pkg.go.dev/github.com/rahmadafandi/fibr/server) — signal-based graceful shutdown via `RunGraceful`, plus `RunGracefulWithConfig` with pre-shutdown hooks and a drain delay.
- [`apierror`](https://pkg.go.dev/github.com/rahmadafandi/fibr/apierror) — typed HTTP errors (`NotFound`, `Conflict`, ...) + a Fiber `ErrorHandler` that renders them as the JSON envelope; wired by `bootstrap` automatically.
- [`webhook`](https://pkg.go.dev/github.com/rahmadafandi/fibr/webhook) — HMAC sign/verify (Stripe-style, replay-protected) + inbound verification middleware.
- [`openapi`](https://pkg.go.dev/github.com/rahmadafandi/fibr/openapi) — generate an OpenAPI 3.0.3 document from registered routes + reflected request/response structs (json + validator tags), served as `/openapi.json` with a CDN-backed Swagger UI at `/docs`. Wired by `bootstrap` via `Options.OpenAPI`.
- [`ws`](https://pkg.go.dev/github.com/rahmadafandi/fibr/ws) — typed WebSocket `Hub[T]` with rooms and JSON broadcast on top of gofiber/contrib/websocket; optional Redis backplane (`WithRedis`) for multi-replica fanout.
- [`sse`](https://pkg.go.dev/github.com/rahmadafandi/fibr/sse) — Server-Sent Events: a one-way `text/event-stream` helper with JSON event encoding.
- [`i18n`](https://pkg.go.dev/github.com/rahmadafandi/fibr/i18n) — dependency-free internationalization: a message `Bundle` (nested JSON, `{placeholder}` substitution, one/other plurals, fallback locale) + locale-detection middleware (query/cookie/Accept-Language) and `T`/`N`/`Locale` helpers. Wired by `bootstrap` via `Options.I18n`.
- [`bootstrap`](https://pkg.go.dev/github.com/rahmadafandi/fibr/bootstrap) — one-call app wiring: middleware, health, DB, metrics, tracing, security headers (helmet), compression, idempotency keys, OpenAPI docs, graceful shutdown.
- [`fibrtest`](https://pkg.go.dev/github.com/rahmadafandi/fibr/fibrtest) — test harness: a fluent HTTP client over `*fiber.App` (`Get`/`Post`/builder), response assertions (`ExpectStatus`/`JSON`), plus `NewDB` (in-memory SQLite Bun) and `Token` (JWT) helpers.
