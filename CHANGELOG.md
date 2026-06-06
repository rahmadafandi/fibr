# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project has not yet had a stable release; all changes are listed under **Unreleased**.

## [Unreleased]

### Added

- `metrics` package: Prometheus request metrics middleware + `/metrics` handler
  (`http_requests_total` / `http_request_duration_seconds` with route-template
  labels; default Go/process collectors). Opt-in via `bootstrap.Options.Metrics`;
  generated apps toggle it with `METRICS_ENABLED`.
- `auth` package: bcrypt `Hash`/`Compare`, JWT bearer middleware
  (`RequireAuth`/`Optional`), claims accessors (`Claims`/`Subject`), and scope
  checks (`RequireScope`/`HasScope`/`Scopes`).
- `auth` refresh tokens + revocation: `Issuer` (`Issue`/`Refresh`/`Logout`) mints
  rotating access+refresh JWT pairs (`TokenPair`) with family-based reuse
  detection; `TokenStore` interface with `RedisStore` and `MemoryStore` impls;
  `WithBlocklist` makes `RequireAuth`/`Optional` reject revoked tokens by `jti`
  (refresh tokens are also rejected for API access).
- `create-fiber-app --auth`: scaffolds an auth module (Account, register/login/me
  + scope-gated route, accounts migration) and generates a random `JWT_SECRET`.
  Login returns an access+refresh `TokenPair`; adds `/auth/refresh` and
  `/auth/logout`; wires a redis-backed token store when `REDIS_URL` is set,
  falling back to an in-memory store otherwise.
- `auth` team/workspace helpers: `ActiveTeam`, `TeamRole`, `RequireTeam`, and
  `RequireRole` read the active team carried in the JWT (`team`/`role` claims);
  the `Issuer` now propagates `team`/`role` through refresh.
- `create-fiber-app --auth-with-team` (implies `--auth`): multi-tenant scaffold
  where one account belongs to many teams via memberships, each team role maps to
  a permission set (carried as the active team's `scopes`). Adds Team/Membership
  entities + migrations, `POST /auth/switch-team`, `GET /teams`, `POST /teams`,
  `POST /teams/:id/members` (gated by `member:manage`), team-aware `/me`, and a
  `team:manage` example route; register auto-creates a personal team.
- `create-fiber-app --auth-with-team` dynamic roles: each team owns its roles in
  the database (`roles` + `role_permissions` tables, seeded with
  owner/admin/member/viewer on team creation) instead of a static code map.
  Permissions are drawn from a fixed code catalog (`GET /permissions`) and
  resolved into the JWT at login/switch-team. Adds role management —
  `GET/POST /teams/:id/roles`, `PUT/DELETE /teams/:id/roles/:name` (owner-
  protected, in-use 409), and `PUT /teams/:id/members` (change a member's role) —
  all gated by `role:manage`/`member:manage` and scoped to the active team.
- `migrate` package: `bun/migrate` wrapper (`Up`/`Down`/`Status`/`Create`) plus a
  ready cobra `NewCommand` (up/down/status/create).
- Generated projects are now a single cobra binary with `serve` + `migrate`
  subcommands and an `internal/migrations/` package; `--sample` and `add module`
  emit `create table` migrations.
- `bootstrap.Options.AutoMigrate` to run module `Migrate` at startup (dev).
- `bootstrap.Module` interface and `App.Mount` for self-wiring feature modules,
  with optional `Migrator` and `HealthChecker` capabilities.
- `health.RegisterProvider` / `RegisterProviderAt` for live readiness checks.
- `create-fiber-app add module <name>` subcommand scaffolding a complete feature
  module for the detected layout (ddd/layered).
- **`database`**: `NewBun` opens a Bun ORM connection with automatic Postgres/SQLite dialect detection from the DSN; pool options (`WithMaxOpenConns`, `WithMaxIdleConns`, `WithConnMaxLifetime`, `WithPingTimeout`, `WithoutPing`).
- **`health`**: `Register`/`RegisterAt` mount `/livez` (liveness) and `/readyz` (readiness) endpoints that run named checks concurrently; `/readyz` applies an overall deadline so a stuck check cannot hang the server. `PingBun` and `Check` helpers provided.
- **`server`**: `RunGraceful` starts a Fiber app and blocks until SIGINT/SIGTERM, then shuts the server down cleanly and calls optional cleanup hooks.
- **`bootstrap`**: `New(Options)` wires recover, request-id, structured logging, optional CORS, optional rate limiting (health probes are exempt), optional DB health checks, and graceful shutdown in a single call. `App.Run(addr)` replaces `fiber.App.Listen`.
- **`config`**: `default:"..."` and `required:"true"` struct tags; extended type support (`time.Duration`, float, bool, integer with per-kind overflow detection, comma-separated `[]string`); combined error reporting across all fields.
- **`http`**: Context-aware JSON client built on fasthttp. All methods (`Get`, `Post`, `Put`, `Patch`, `Delete`) accept `context.Context` and return `(statusCode int, err error)`. `WithRetry` retries only on 5xx; `WithTimeout`, `WithHeader` options; `FireAndForget` for background non-blocking calls; `HTTPError` type for status-code errors.
- **`redis`**: Generic `Remember` cache-aside helper that returns a cached value or calls a loader on miss. `ParseRedisOptions` parses a Redis URL into `*redis.Options` and returns an error on bad input.
- **`validator`**: `Register` for custom validation rules; error messages use JSON field names and include the failing value; handles non-struct input without panicking.
- **`jwt`**: `GenerateTokenWithExpiry` for generating tokens with an explicit expiry duration.
- **`uploader`**: `WithMaxSize` and `WithAllowedMime` options for `NewLocalUploader`; filenames are sanitized and permissions are tightened; partial files are removed on write error.
- **`middleware`**: `X-Request-ID` response header added by the request-id middleware so callers can correlate responses.
- **`context`**: `SetLocal` and `GetLocal` typed helpers replace the former `CustomContext`.
- **`pagination`**: `pagination` package extracted to manage page/limit logic independently.

### Changed

- `bootstrap.App.Mount` runs a module's `Migrate` only when the app was built
  with `AutoMigrate: true` (previously always). Schema is owned by migrations by
  default.
- `create-fiber-app --sample` now generates the sample `user` feature as a
  `bootstrap.Module` mounted via `app.Mount` (previously hand-wired in main.go).
- **ORM migration**: GORM removed; Bun is now the ORM throughout the library. `parser` pagination scopes and `slug.Generate` (now `slug.Generate(ctx, db, table, text) (string, error)`) migrated accordingly.
- **`config.LoadConfig`**: Signature changed from a generic return (`LoadConfig[T]() (T, error)`) to pointer-out (`LoadConfig(out any) error`), matching idiomatic Go.
- **`http` methods**: All request methods now accept a leading `context.Context` argument and return `(int, error)` instead of a plain error.
- **`common` package removed**: Functionality split into the `response` and `logger` packages.

### Removed

- **`common` package**: Replaced by dedicated `response` and `logger` packages.
- **`parser.ParseBody` / `ParseQuery` / `ParseParams`**: Removed generic parse helpers; use Fiber's built-in `c.BodyParser`, `c.QueryParser`, and `c.ParamsParser` directly.
- **`redis.GormResult`**: Removed broken GORM-coupled helper; use `Remember` instead.

### Fixed

- **`pagination`**: Guard against divide-by-zero when `limit` is zero; clamp page number to a valid range.
- **`parser`**: Corrected ILIKE pattern construction; added SQL-injection guard on the sort column expression.
- **`uploader`**: Path-traversal protection on uploaded filenames; partial file is cleaned up on write error; file permissions tightened.
- **`validator`**: No longer panics on non-struct input.
- **`config`**: Per-kind numeric overflow detection; errors from all fields are combined and reported together.
- **`slug`**: Uniqueness retry loop is now capped to prevent an infinite spin.
- **`redis`**: `ParseRedisOptions` returns an error instead of `nil` on a bad URL.
- **`server`**: Signal-notify goroutine and exit bridge goroutine are cleaned up correctly on return.
- **`database`**: Malformed DSN schemes are rejected with an error instead of being silently treated as SQLite.
- **`http`**: `FireAndForget` detaches from the caller's context so cancellation does not abort background requests; retry backoff is context-aware.
