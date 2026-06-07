# Contributing

Thanks for your interest in improving fiber-helpers!

## Development

Requires Go 1.26+.

```bash
git clone https://github.com/rahmadafandi/fiber-helpers
cd fiber-helpers
go build ./...
go test ./...
```

End-to-end generator tests (slower; they generate and compile apps) are gated:

```bash
RUN_E2E=1 go test ./cmd/create-fiber-app/
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
