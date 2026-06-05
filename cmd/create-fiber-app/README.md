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
| `--auth` | `false` | scaffold JWT auth (Account model, register/login/me routes) |
| `--no-git` | `false` | skip `git init` |
| `--no-tidy` | `false` | skip `go mod tidy` |
| `--helpers-version` | `latest` | fiber-helpers version pinned in go.mod |
| `--local` | | replace fiber-helpers with a local path (development) |

## Adding a module

Inside a generated project, scaffold a new feature module (model, repository,
service, handler, and the wiring that implements `bootstrap.Module`):

```bash
create-fiber-app add module product
```

The layout (`ddd`/`layered`) is auto-detected from the project; override with
`--layout`. Use `--dir` to target a project other than the current directory.
The command prints the `app.Mount(...)` line (and import) to paste into
`cmd/api/main.go`. It refuses to overwrite existing module files.

It also generates a `create table` migration in `internal/migrations/`; run
`go run ./cmd/api migrate up` to apply it.

Field note: scaffolded entities have `ID` and `Name`; rename/extend them for
your domain. Pluralization for the table/route is naive (`product` → `products`).

## Migrations

Generated projects are a single binary. `./app` (or `go run ./cmd/api`) runs the
server; migrations are subcommands:

```bash
go run ./cmd/api migrate up        # apply pending migrations
go run ./cmd/api migrate down      # roll back the last group
go run ./cmd/api migrate status    # show applied/pending
go run ./cmd/api migrate create x  # scaffold internal/migrations/<ts>_x.go
```

`--sample` and `add module` generate a `create table` migration under
`internal/migrations/`. For a fast dev loop you can skip migrations and create
tables at startup with `AUTO_MIGRATE=true ./app` (or `./app serve --auto-migrate`).
Makefile targets: `make migrate-up`, `migrate-down`, `migrate-status`,
`migrate-create name=add_foo`.

## Auth

`--auth` scaffolds JWT authentication: an `Account` model (email + bcrypt
password + comma-separated scopes), an `accounts` migration, and `/auth` routes —
`POST /auth/register`, `POST /auth/login` (returns a JWT), `GET /auth/me`
(protected), and `GET /auth/admin` (requires the `admin` scope). A random
`JWT_SECRET` is generated into `.env.example`; override it per environment. Works
alongside `--sample` (accounts and the demo users table are independent).

## Note

Generated projects depend on `github.com/rahmadafandi/fiber-helpers`. Until that module is published to a proxy, use `--local /path/to/fiber-helpers` (adds a `replace`) or `--no-tidy` and wire the dependency manually.
