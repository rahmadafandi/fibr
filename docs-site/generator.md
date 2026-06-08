# fibr

Scaffolds a batteries-included Fiber project (a cobra single binary with `serve`,
`migrate`, and — with `--queue` — a `worker` subcommand).

## Install

```bash
go install github.com/rahmadafandi/fibr/cmd/fibr@latest
```

## Usage

```bash
fibr new <name> --module <module-path> [flags]
```

| Flag | Description |
|------|-------------|
| `--module` | Go module path (required) |
| `--db` | `postgres` (default) or `sqlite` |
| `--layout` | `ddd` (default) or `layered` |
| `--sample` | include a sample CRUD module (`user`) |
| `--auth` | JWT auth + accounts: register/login/me, refresh/logout, bcrypt, scopes |
| `--auth-with-team` | teams/workspaces, per-team roles, invitations, admin (implies `--auth`) |
| `--queue` | background job queue (asynq) + asynqmon UI + `worker` subcommand |
| `--mailer` | transactional email (SMTP), wired into invitations and the welcome job |
| `--dir` | output directory (default `./<name>`) |
| `--local` | replace fibr with a local path (unpublished-library development) |

Run without flags to be prompted interactively.

## Adding modules

```bash
fibr add module product
```

Generates a Mount-based module and prints the `app.Mount(...)` line to wire it in.

## Configuration

Generated apps read environment variables (see the generated `.env.example`):
`DATABASE_URL`, `PORT`, `AUTO_MIGRATE`, and — depending on the flags used —
`JWT_SECRET`, `REDIS_URL`, `METRICS_ENABLED`, `TRACING_ENABLED`,
`QUEUE_CONCURRENCY`, `ASYNQMON_PATH`, and `SMTP_*`.
