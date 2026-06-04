# Fiber Helpers

A collection of helper packages for the [Fiber](https://gofiber.io/) web framework.

## Packages

### `config`

Loads configuration from environment variables (and a `.env` file if present) into a typed struct. Supports `default:"..."` and `required:"true"` tags, as well as `time.Duration`, float, bool, integer, and comma-separated string-slice fields.

**Usage:**

```go
import "github.com/rahmadafandi/fiber-helpers/config"

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
import "github.com/rahmadafandi/fiber-helpers/logger"

log := logger.Default()
log.Info("Hello, world!")
```

### `response`

Helper functions for sending standardized JSON responses.

**Usage:**

```go
import "github.com/rahmadafandi/fiber-helpers/response"

response.SendSuccess(c, data, "Success")
response.SendError(c, nil, "Error", 400)
```

### `parser`

Helper functions for pagination with Bun. The `Paginate` helper returns a Bun query modifier for use with `query.Apply`.

Request body/query/params parsing uses Fiber's built-in `c.BodyParser(&out)`,
`c.QueryParser(&out)`, and `c.ParamsParser(&out)` directly.

**Usage:**

```go
import "github.com/rahmadafandi/fiber-helpers/parser"

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

### `validator`

A helper package for validating structs using [go-playground/validator](https://github.com/go-playground/validator). Supports custom validation rules via `Register`.

**Usage:**

```go
import "github.com/rahmadafandi/fiber-helpers/validator"

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

### `jwt`

A helper package for working with JSON Web Tokens.

**Usage:**

```go
import "github.com/rahmadafandi/fiber-helpers/jwt"

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
    fhttp "github.com/rahmadafandi/fiber-helpers/http"
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
    firedis "github.com/rahmadafandi/fiber-helpers/redis"
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

### `slug`

Generates a unique, URL-safe slug for a given table using a [Bun](https://bun.uptrace.dev/) database.

**Usage:**

```go
import (
    "context"
    "github.com/rahmadafandi/fiber-helpers/slug"
)

// Returns e.g. "my-first-post-abc123defgh456"
s, err := slug.Generate(ctx, db, "posts", "My First Post")
```

### `uploader`

A helper package for uploading files to local storage. `NewLocalUploader` accepts functional options for max file size and MIME type allowlist. The filename is sanitized automatically.

**Usage:**

```go
import "github.com/rahmadafandi/fiber-helpers/uploader"

// Create a local uploader (max 5 MB, images only)
up := uploader.NewLocalUploader("./uploads",
    uploader.WithMaxSize(5<<20),
    uploader.WithAllowedMime([]string{"image/jpeg", "image/png"}),
)

// Upload a file (filename is sanitized before saving)
path, err := up.Upload(file, filename)
```

### `middleware`

A collection of useful middleware.

**Usage:**

```go
import "github.com/rahmadafandi/fiber-helpers/middleware"

app := fiber.New()

// Recover from panics
app.Use(middleware.Recover(logger))

// Log requests
app.Use(middleware.RequestLogger(logger))

// Protect routes
app.Use(middleware.Auth(secret))

// Context
app.Use(middleware.ContextMiddleware(10 * time.Second))
```

### `database`

Opens a [Bun](https://bun.uptrace.dev/) database, picking the dialect from the DSN
(`postgres://` → Postgres, `file:`/`:memory:`/path → SQLite).

```go
import "github.com/rahmadafandi/fiber-helpers/database"

db, err := database.NewBun("postgres://localhost/app",
    database.WithMaxOpenConns(20),
    database.WithPingTimeout(3*time.Second),
)
```

### `health`

Liveness (`/livez`) and readiness (`/readyz`) endpoints with concurrent checks.

```go
import "github.com/rahmadafandi/fiber-helpers/health"

health.Register(app, health.PingBun(db),
    health.Check("cache", func(ctx context.Context) error { return rds.Ping(ctx) }),
)
// GET /livez  -> 200 {"status":"ok"}
// GET /readyz -> 200/503 {"status":"...","checks":{...}}
```

### `server`

Signal-based graceful shutdown.

```go
import "github.com/rahmadafandi/fiber-helpers/server"

err := server.RunGraceful(app, ":3000", 10*time.Second, func(ctx context.Context) error {
    return db.Close()
})
```

### `bootstrap`

Optional one-call wiring of recover, request id, logging, optional CORS / rate
limit / health, and graceful shutdown.

```go
import "github.com/rahmadafandi/fiber-helpers/bootstrap"

app := bootstrap.New(bootstrap.Options{
    DB:           db,
    EnableCORS:   true,
    RateLimit:    100,
    HealthChecks: []health.NamedCheck{health.PingBun(db)},
})
app.Get("/", handler)
log.Fatal(app.Run(":3000")) // graceful shutdown + db.Close handled
```
