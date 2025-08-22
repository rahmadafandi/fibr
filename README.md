# Fiber Helpers

A collection of helper packages for the [Fiber](https://gofiber.io/) web framework.

## Packages

### `logger`

A structured logger based on [zerolog](https://github.com/rs/zerolog).

**Usage:**

```go
import "github.com/yomaaf/fiber-helpers/logger"

log := logger.Default()
log.Info("Hello, world!")
```

### `response`

Helper functions for sending standardized JSON responses.

**Usage:**

```go
import "github.com/yomaaf/fiber-helpers/response"

response.SendSuccess(c, data, "Success")
response.SendError(c, nil, "Error", 400)
```

### `parser`

Helper functions for parsing request body, query, and params.

**Usage:**

```go
import "github.com/yomaaf/fiber-helpers/parser"

type MyStruct struct {
    Name string `json:"name"`
}

// Parse body
body, err := parser.ParseBody[MyStruct](c)

// Parse query
query, err := parser.ParseQuery[MyStruct](c)

// Parse params
params, err := parser.ParseParams[MyStruct](c)
```

### `validator`

A helper package for validating structs using [go-playground/validator](https://github.com/go-playground/validator).

**Usage:**

```go
import "github.com/yomaaf/fiber-helpers/validator"

type MyStruct struct {
    Name string `json:"name" validate:"required"`
}

var body MyStruct

if errs := validator.ValidateStruct(&body); len(errs) > 0 {
    // Handle validation errors
}
```

### `jwt`

A helper package for working with JSON Web Tokens.

**Usage:**

```go
import "github.com/yomaaf/fiber-helpers/jwt"

// Generate a token
token, err := jwt.GenerateToken(claims, secret)

// Validate a token
valid, err := jwt.ValidateToken(token, secret)
```

### `uploader`

A helper package for uploading files.

**Usage:**

```go
import "github.com/yomaaf/fiber-helpers/uploader"

// Create a local uploader
up := uploader.NewLocalUploader("./uploads")

// Upload a file
path, err := up.Upload(file)
```

### `middleware`

A collection of useful middleware.

**Usage:**

```go
import "github.com/yomaaf/fiber-helpers/middleware"

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
