# Examples

Runnable programs showing how to wire fibr together. This is a separate
Go module (`replace`-d to the parent), so run the commands from this directory.

| Command | What it shows | Needs |
|---------|---------------|-------|
| [`go run .`](main.go) | Minimal bootstrap: DB, health, CORS, rate limit, one route | — |
| [`go run ./auth`](auth/main.go) | Password hashing + JWT access/refresh issuer (rotate, reuse-detect, logout) | — |
| [`go run ./mailer`](mailer/main.go) | Render an HTML/text email from templates and send it | — |
| [`go run ./queue`](queue/main.go) | Enqueue a job and process it with an asynq worker | Redis |

For the queue example, start Redis first:

```bash
docker run --rm -p 6379:6379 redis
REDIS_URL=redis://localhost:6379 go run ./queue
```

For per-function API snippets, see the runnable godoc examples on
[pkg.go.dev](https://pkg.go.dev/github.com/rahmadafandi/fibr)
(packages `pagination`, `jwt`, `validator`, `parser`, `response`, `slug`).
