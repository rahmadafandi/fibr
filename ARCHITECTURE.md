# Architecture

> A map of how **fibr** (`github.com/rahmadafandi/fibr`) is put together: the
> packages, how they layer, and how `bootstrap` composes them into an app.
> Derived from a dependency-graph analysis of the codebase (30 library packages
> plus the `fibr` generator, Go 1.26, Fiber v2).

## What fibr is

fibr is a toolkit for building [Fiber](https://github.com/gofiber/fiber) HTTP
services, plus a code generator (`fibr new`) that scaffolds a runnable app from
those packages. Two usage modes:

1. **Library** — import any package on its own (`config`, `auth`, `redis`, `ws`, …).
   Each stands alone with a small, explicit API.
2. **One-call wiring** — `bootstrap.New(Options{...})` composes the packages into
   a configured `*fiber.App` (middleware, health, metrics, tracing, docs, …),
   every feature an opt-in `Options` field.

## Design principles

- **Standalone helpers, one composer.** Every package is independently usable and
  one-way in its dependencies. `bootstrap` is the single integration point that
  wires them together — nothing depends on `bootstrap` except the generated app.
- **Opt-in everything.** Features are off unless an `Options` field turns them on.
  No hidden global state.
- **Generics for type-safety.** `bind.Body[T]`, `jobs.Handle[T]`, `redis.Subscribe[T]`,
  `ws.Hub[T]` carry the payload type instead of `interface{}`.
- **Structural interfaces over imports.** Adapters satisfy `fiber.Storage` and
  similar interfaces by shape, so leaf packages avoid importing heavy deps.

## Layering

Dependencies flow one way. Leaf packages (left) never import the layers above them.

```
external            fibr leaf            fibr mid              composer
────────            ─────────            ────────              ────────
jwt (golang-jwt) ─► jwt        ─────────► auth ──────────┐
bun (uptrace)    ─► database, parser,                     │
                    slug, migrate                         │
go-redis         ─► redis ──────────────► ws (backplane), ├─► bootstrap ─► (generated app)
                                          auth (blocklist),│      cmd/fibr
fiber, fasthttp  ─► (all http-facing pkgs)                │
                    apierror ◄────────────(every handler) ┘
```

Key one-way edges (verified in the graph):

- `auth` → `jwt` — `auth` is the policy layer (bearer middleware, `Issuer`,
  token blocklist); `jwt` is the crypto primitive (sign/verify/claims). `jwt` has
  zero dependencies on `auth`.
- `database`, `parser`, `slug`, `migrate` → **Bun ORM** — the data layer.
- `ws`, `auth`, rate-limit storage, idempotency storage → `redis` — shared infra
  (pub/sub backplane, token blocklist, `fiber.Storage` adapters).
- every request handler → `apierror` — typed errors funnel through one
  `ErrorHandler` so responses are uniform.

## Package catalog

### Core HTTP & app lifecycle
| Package | Responsibility |
|---|---|
| `bootstrap` | One-call app wiring; composes everything below (see *Composition*). |
| `middleware` | Recover, request logging, request-id/context, auth guard. |
| `context` | Request context, request-id, type-safe locals (`GetLocal[T]`). |
| `response` | Standardized JSON success/error envelope. |
| `apierror` | Typed HTTP errors (`NotFound`, `Conflict`, …) + a Fiber `ErrorHandler`. |
| `health` | Liveness (`/livez`) + readiness (`/readyz`) endpoints. |
| `server` | Signal-based graceful shutdown (`RunGraceful`). |

### Config & observability
| Package | Responsibility |
|---|---|
| `config` | Env → typed struct via `default`/`required` tags. |
| `logger` | Structured logging (zerolog). |
| `metrics` | Prometheus request metrics + `/metrics`. |
| `tracing` | OpenTelemetry (OTLP/HTTP) setup + Fiber spans. |

### Data
| Package | Responsibility |
|---|---|
| `database` | Bun connector, Postgres/SQLite auto-detect, optional tracing. |
| `migrate` | Versioned migrations + a ready cobra command. |
| `parser` | Bun pagination/search query modifiers. |
| `pagination` | Paginated result envelope with page metadata. |
| `slug` | Unique URL-safe slug generator backed by Bun. |

### Request handling & security
| Package | Responsibility |
|---|---|
| `bind` | Parse + validate body/query/params into `T` (`Body[T]`/`Query[T]`/`Params[T]`). |
| `validator` | Struct validation with custom rules + JSON field names. |
| `auth` | JWT bearer auth, bcrypt, refresh-token rotation, scopes, teams/roles, revocation store. |
| `jwt` | JWT generation/validation primitives. |
| `webhook` | HMAC sign/verify (Stripe-style, replay-protected) + inbound middleware. |

### Integrations
| Package | Responsibility |
|---|---|
| `redis` | go-redis wrapper: `Remember` cache-aside, `Storage` (fiber.Storage) adapter, `Publish`/`Subscribe[T]`. |
| `jobs` | Redis-backed background jobs (asynq) + `Scheduler` (cron) + asynqmon mount. |
| `mailer` | Transactional email: pluggable `Sender` (SMTP/log/memory) + templates. |
| `uploader` | Local + S3-compatible (`S3Uploader`) file upload with size/MIME limits. |
| `http` | Context-aware JSON HTTP client with retry. |

### Realtime, docs, i18n
| Package | Responsibility |
|---|---|
| `ws` | Typed WebSocket `Hub[T]` (rooms, broadcast, per-conn write pump, ping) + optional Redis backplane for multi-replica fanout. |
| `sse` | Server-Sent Events stream helper (`Handler` + `Stream.Send`/`Event`). |
| `openapi` | OpenAPI 3.0.3 generation from registered routes + reflected structs; serves `/openapi.json` + Swagger UI `/docs`. |
| `i18n` | Dependency-free message bundle (nested JSON, placeholders, plurals, fallback) + locale-detection middleware (`T`/`N`/`Locale`). |

### Generator
| Package | Responsibility |
|---|---|
| `cmd/fibr` | `fibr new` — scaffolds a runnable app (ddd or layered layout) from the packages above, gated by flags (`--auth`, `--queue`, `--mailer`, `--realtime`, `--i18n`, …). |

## Composition: `bootstrap.New`

`bootstrap.New` is the architectural hub — it pulls in ~18 packages and turns a
flat `Options` struct into a wired `*fiber.App`. Order matters:

**Request pipeline** (`f.Use`, in order):
1. `middleware.Recover` — panic guard
2. `middleware.ContextMiddleware` — request-id, context, timeout
3. `i18n.Middleware` *(if `Options.I18n`)*
4. `helmet` *(if `SecurityHeaders`)*
5. `compress` *(if `Compression`)*
6. `idempotency` *(if `Idempotency`)* — backed by `IdempotencyStorage`
7. `otelfiber` *(if `Tracing`)*
8. `middleware.RequestLogger`
9. `metrics.Middleware` *(if `Metrics`)*
10. `cors` *(if `EnableCORS`)*
11. `limiter` *(if `RateLimit > 0`)* — backed by `RateLimitStorage`

**Error handling:** `apierror.Handler` is installed as the default
`FiberConfig.ErrorHandler`, so every package's typed errors render as the same
JSON envelope.

**Mounted endpoints:**
- `health` provider → `/livez`, `/readyz` (registered **before** the rate limiter so
  probes are never throttled)
- `/metrics` *(if `Metrics`)*
- asynqmon UI *(if `Asynqmon` set)*
- `/openapi.json` + `/docs` *(if `OpenAPI` set)*

**Lifecycle:** DB close + caller `Cleanup` hooks run on graceful shutdown via
`server.RunGraceful`.

## Cross-cutting concerns

- **Error funnel.** Handlers return `*apierror.Error`; `bootstrap` wires
  `apierror.Handler` once. One place defines the wire format.
- **Shared Redis.** A generated app opens **one** go-redis client and reuses it for
  rate-limit storage, idempotency storage, the WebSocket backplane, and the auth
  token store. (The asynq job queue manages its own connection pool separately.)
- **Response envelope.** `response` + `pagination` define the success/list shapes;
  `apierror` defines the error shape.

## Generated apps (`fibr new`)

The generator renders one of two layouts and wires the same packages:

- **ddd** — `internal/domain`, `internal/application`, `internal/infrastructure`,
  `internal/interface/http`.
- **layered** — `internal/handler`, `internal/service`, `internal/repository`,
  `internal/router`.

Feature flags add scaffolding: `--auth`/`--auth-with-team` (accounts, teams,
roles, invitations), `--queue` (worker + scheduler commands), `--mailer`,
`--realtime` (WebSocket chat + SSE route), `--i18n` (en/id catalogs). Generated
apps expose `/openapi.json` + `/docs` out of the box. End-to-end tests compile
every flag combination.

## Testing & CI

- Unit + example tests per package; `_test.go` helpers form their own dense
  cluster in the dependency graph (expected for a library).
- CI (GitHub Actions): `test` (race + coverage), `lint` (golangci-lint),
  `e2e` (generate + compile apps), `vuln` (govulncheck), `examples`, CodeQL.
- Local pre-commit/pre-push hooks (lefthook) mirror the CI lint/test gates.
- Releases cut via goreleaser on `v*` tags; docs published with mkdocs.

---

*This document is a hand-curated summary of an automated dependency-graph analysis
(communities ≈ one per package; `bootstrap` as the composition hub). Update it when
package boundaries or the `bootstrap` wiring change.*
