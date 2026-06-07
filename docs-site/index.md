# fiber-helpers

A collection of helper packages for the [Fiber](https://gofiber.io/) web
framework, plus **create-fiber-app** — a generator that scaffolds
batteries-included Fiber projects.

Requires Go 1.26+. Targets Fiber v2 and Bun ORM (Postgres or SQLite).

## Install

The library:

```bash
go get github.com/rahmadafandi/fiber-helpers
```

The project generator:

```bash
go install github.com/rahmadafandi/fiber-helpers/cmd/create-fiber-app@latest
```

## Quickstart

Scaffold and run a new app:

```bash
create-fiber-app myapp --module example.com/myapp --db sqlite --layout ddd
cd myapp
go run ./cmd/api migrate up   # create tables
go run ./cmd/api              # serve on :3000
```

Add features with flags — `--auth`, `--auth-with-team`, `--queue`, `--mailer`,
`--sample`. See the [Generator](generator.md) guide.

## Documentation

- [Generator](generator.md) — all `create-fiber-app` options.
- [Packages](packages.md) — the helper packages.
- Full API reference:
  [pkg.go.dev/github.com/rahmadafandi/fiber-helpers](https://pkg.go.dev/github.com/rahmadafandi/fiber-helpers)
