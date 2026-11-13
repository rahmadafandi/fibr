# Migrating from fibr v1 to v2

fibr v2 targets **Fiber v3** and cleans up a few APIs. v1 (Fiber v2) remains
available at `github.com/rahmadafandi/fibr`; v2 lives at a new module path, so
both can be used side by side while you migrate.

## 1. Update the module path

```bash
go get github.com/rahmadafandi/fibr/v2
```

Replace every import of `github.com/rahmadafandi/fibr/...` with
`github.com/rahmadafandi/fibr/v2/...`.

## 2. Move to Fiber v3

fibr v2 requires `github.com/gofiber/fiber/v3`. Fiber's own
[migration guide](https://docs.gofiber.io/whats_new) covers the framework
changes in full; the ones you will hit when using fibr are:

| Fiber v2 | Fiber v3 |
| --- | --- |
| `func(c *fiber.Ctx) error` | `func(c fiber.Ctx) error` |
| `c.UserContext()` / `c.SetUserContext(ctx)` | `c.Context()` / `c.SetContext(ctx)` |
| `c.Context()` (fasthttp) | `c.RequestCtx()` |
| `c.BodyParser(&v)` | `c.Bind().Body(&v)` |
| `c.QueryParser(&v)` | `c.Bind().Query(&v)` |
| `c.ParamsParser(&v)` with `params:"id"` tags | `c.Bind().URI(&v)` with `uri:"id"` tags |
| `app.Test(req, 5000)` | `app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})` |
| `cors.Config{AllowOrigins: "https://a.com"}` | `cors.Config{AllowOrigins: []string{"https://a.com"}}` |

fibr's own helpers (`bind`, `reqctx`, `fibrtest`, `bootstrap`) already use the
v3 forms, so code that goes through them needs only the handler signature
change.

## 3. Renamed packages

Three packages were renamed so they no longer collide with packages they are
imported next to. Drop the import alias you were using and rename the
qualifier; the APIs are unchanged.

| v1 import | v2 import |
| --- | --- |
| `fibr/context` | `fibr/v2/reqctx` |
| `fibr/http` | `fibr/v2/httpclient` |
| `fibr/redis` | `fibr/v2/rediskit` |

```go
// v1
import fhredis "github.com/rahmadafandi/fibr/redis"
rds := fhredis.New(client)

// v2
import "github.com/rahmadafandi/fibr/v2/rediskit"
rds := rediskit.New(client)
```

## 4. Safer defaults

These behaviors were opt-in in v1.6 and are the default in v2.

### Uploads get unique names

`Upload` stores each file as `<random>-<name>` and takes a `context.Context`.

```go
// v1
path, err := up.Upload(file, "avatar.png")            // ./uploads/avatar.png, overwrites

// v2
path, err := up.Upload(ctx, file, "avatar.png")       // ./uploads/3f9c…-avatar.png
```

- To keep the old naming (and overwrite behavior), construct the uploader with
  `uploader.WithOriginalNames()`.
- `WithUniqueNames`, `UploadContext`, and `ContextUploader` are removed.
- If you stored the returned path or key, nothing changes for existing files;
  only new uploads get the prefixed name.

### Tokens must expire

`auth.RequireAuth` and `auth.Optional` reject tokens without an `exp` claim.
Tokens from `auth.Issuer` and `jwt.GenerateTokenWithExpiry` are unaffected. If
you mint non-expiring tokens with `jwt.GenerateToken` on purpose, pass
`auth.WithExpirationOptional()` to the middleware. `WithExpirationRequired` is
removed.

### CORS is explicit

`bootstrap.Options.EnableCORS` is removed, because it allowed every origin.

```go
// v1
bootstrap.Options{EnableCORS: true}

// v2: list the origins…
bootstrap.Options{CORSOrigins: []string{"https://app.example.com"}}

// …or pass a full config
bootstrap.Options{CORS: &cors.Config{AllowOrigins: []string{"https://app.example.com"}, AllowCredentials: true}}
```

With neither set, CORS is off. To deliberately allow every origin, say so:
`CORS: &cors.Config{AllowOrigins: []string{"*"}}`.

## 5. Other changes

- `redis.Storage` (now `rediskit.Storage`) implements Fiber v3's
  `fiber.Storage`, adding `GetWithContext`, `SetWithContext`,
  `DeleteWithContext`, and `ResetWithContext`.
- Tracing uses `github.com/gofiber/contrib/v3/otel`; `ws` uses
  `github.com/gofiber/contrib/v3/websocket`. If you import the contrib
  packages yourself, switch to the `v3` paths.

## Generated apps

`fibr new` (v2) generates Fiber v3 apps. An app generated with v1 is your own
code and keeps working on fibr v1; to move it to v2, apply the steps above. New
apps read allowed CORS origins from `CORS_ALLOW_ORIGINS` (comma-separated).
