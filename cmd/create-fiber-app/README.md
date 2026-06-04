# create-fiber-app

Generate a ready-to-run [Fiber](https://gofiber.io/) project wired with [fiber-helpers](https://github.com/rahmadafandi/fiber-helpers).

## Install

```bash
go install github.com/rahmadafandi/fiber-helpers/cmd/create-fiber-app@latest
```

## Usage

```bash
create-fiber-app myapp --module github.com/me/myapp --db postgres --layout ddd
```

Run with no flags for an interactive wizard. Flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--module` | (required) | Go module path |
| `--db` | `postgres` | `postgres` or `sqlite` |
| `--layout` | `ddd` | `ddd` or `layered` |
| `--sample` | `false` | include a sample CRUD domain |
| `--dir` | `./<name>` | output directory |
| `--no-git` | `false` | skip `git init` |
| `--no-tidy` | `false` | skip `go mod tidy` |
| `--helpers-version` | `latest` | fiber-helpers version pinned in go.mod |
| `--local` | | replace fiber-helpers with a local path (development) |

## Note

Generated projects depend on `github.com/rahmadafandi/fiber-helpers`. Until that module is published to a proxy, use `--local /path/to/fiber-helpers` (adds a `replace`) or `--no-tidy` and wire the dependency manually.
