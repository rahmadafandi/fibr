# Contributing

Thanks for your interest in improving fibr!

## Development

Requires Go 1.26+.

```bash
git clone https://github.com/rahmadafandi/fibr
cd fibr
go build ./...
go test ./...
```

End-to-end generator tests (slower; they generate and compile apps) are gated:

```bash
RUN_E2E=1 go test ./cmd/fibr/
```

Integration tests run against a real Postgres and Redis and are skipped unless
both are configured. Point them at a disposable database — the tests create and
drop the `outbox`, `outbox_dead`, `inbox`, and `articles` tables:

```bash
FIBR_TEST_POSTGRES_DSN='postgres://user:pass@127.0.0.1:5432/fibr_test?sslmode=disable' \
FIBR_TEST_REDIS_URL='redis://127.0.0.1:6379/1' \
make integration
```

## Before opening a pull request

- `go build ./...`, `go vet ./...`, and `go test ./...` pass.
- `golangci-lint run ./...` is clean (config in `.golangci.yml`).
- Code is `gofmt`/`goimports` formatted.
- Add or update tests for behavior changes.
- Update `CHANGELOG.md` (under `## [Unreleased]`) and any relevant docs.

## Commit messages

Conventional-commit style is preferred: `feat:`, `fix:`, `docs:`, `test:`,
`chore:`, `ci:`, etc.

## Reporting bugs / requesting features

Use the issue templates. Include the version, Go version, layout/flags (for
generator issues), and a minimal reproduction where possible.
