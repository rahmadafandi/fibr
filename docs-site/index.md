# fibr

[![Release](https://img.shields.io/github/v/release/rahmadafandi/fibr)](https://github.com/rahmadafandi/fibr/releases/latest)
[![ci](https://github.com/rahmadafandi/fibr/actions/workflows/ci.yml/badge.svg)](https://github.com/rahmadafandi/fibr/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/rahmadafandi/fibr/branch/master/graph/badge.svg)](https://codecov.io/gh/rahmadafandi/fibr)
[![Go Reference](https://pkg.go.dev/badge/github.com/rahmadafandi/fibr.svg)](https://pkg.go.dev/github.com/rahmadafandi/fibr)
[![Go Report Card](https://goreportcard.com/badge/github.com/rahmadafandi/fibr)](https://goreportcard.com/report/github.com/rahmadafandi/fibr)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/rahmadafandi/fibr/blob/master/LICENSE)

A collection of helper packages for the [Fiber](https://gofiber.io/) web
framework, plus **fibr** — a generator that scaffolds
batteries-included Fiber projects.

Requires Go 1.26+. Targets Fiber v2 and Bun ORM (Postgres or SQLite).

## Install

The library:

```bash
go get github.com/rahmadafandi/fibr
```

The project generator:

```bash
go install github.com/rahmadafandi/fibr/cmd/fibr@latest
```

## Quickstart

Scaffold and run a new app:

```bash
fibr new myapp --module example.com/myapp --db sqlite --layout ddd
cd myapp
go run ./cmd/api migrate up   # create tables
go run ./cmd/api              # serve on :3000
```

Add features with flags — `--auth`, `--auth-with-team`, `--queue`, `--mailer`,
`--sample`. See the [Generator](generator.md) guide.

## Documentation

- [Generator](generator.md) — all `fibr` options.
- [Packages](packages.md) — the helper packages.
- Full API reference:
  [pkg.go.dev/github.com/rahmadafandi/fibr](https://pkg.go.dev/github.com/rahmadafandi/fibr)
