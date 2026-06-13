// Copyright 2026 Rahmad Afandi. MIT License.

// Package database connects to Postgres or SQLite via Bun, with dialect auto-detection and optional tracing.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bunotel"
	"github.com/uptrace/bun/schema"
)

type dialectKind int

const (
	dialectUnknown dialectKind = iota
	dialectPostgres
	dialectSQLite
)

type config struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	pingTimeout     time.Duration
	skipPing        bool
	tracing         bool
}

// Option configures the database connection.
type Option func(*config)

// WithMaxOpenConns sets the maximum number of open connections to the database.
func WithMaxOpenConns(n int) Option { return func(c *config) { c.maxOpenConns = n } }

// WithMaxIdleConns sets the maximum number of idle connections in the pool.
func WithMaxIdleConns(n int) Option { return func(c *config) { c.maxIdleConns = n } }

// WithConnMaxLifetime sets the maximum amount of time a connection may be reused.
func WithConnMaxLifetime(d time.Duration) Option { return func(c *config) { c.connMaxLifetime = d } }

// WithPingTimeout sets the timeout for the connect-time ping (default 5 s).
func WithPingTimeout(d time.Duration) Option { return func(c *config) { c.pingTimeout = d } }

// WithoutPing skips the connect-time ping.
func WithoutPing() Option { return func(c *config) { c.skipPing = true } }

// WithTracing installs Bun's OpenTelemetry query hook so each query is recorded
// as a span on the active trace. It requires a configured tracer provider (see
// the tracing package's Setup); without one, spans are dropped.
func WithTracing() Option { return func(c *config) { c.tracing = true } }

// detectDialect inspects the DSN scheme to choose a Bun dialect. A "postgres://"
// or "postgresql://" scheme is Postgres; "file:", ":memory:", or a bare path is
// SQLite; any other scheme is an error.
func detectDialect(dsn string) (dialectKind, error) {
	switch {
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return dialectPostgres, nil
	case strings.HasPrefix(dsn, "file:"), dsn == ":memory:":
		return dialectSQLite, nil
	default:
		// A bare filesystem path (e.g. "app.db", "/var/data/app.db") is SQLite.
		// Anything else containing a colon is a malformed/unsupported scheme.
		if strings.Contains(dsn, ":") {
			return dialectUnknown, fmt.Errorf("database: unsupported DSN scheme in %q", dsn)
		}
		return dialectSQLite, nil
	}
}

// NewBun opens a Bun database, choosing the dialect from the DSN scheme, applies
// pool options, and (unless WithoutPing) verifies the connection with a ping.
func NewBun(dsn string, opts ...Option) (*bun.DB, error) {
	cfg := config{pingTimeout: 5 * time.Second}
	for _, o := range opts {
		o(&cfg)
	}

	kind, err := detectDialect(dsn)
	if err != nil {
		return nil, err
	}

	var sqldb *sql.DB
	var dialect schema.Dialect

	switch kind {
	case dialectPostgres:
		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		dialect = pgdialect.New()
	case dialectSQLite:
		sqldb, err = sql.Open(sqliteshim.ShimName, dsn)
		if err != nil {
			return nil, fmt.Errorf("database: open sqlite: %w", err)
		}
		dialect = sqlitedialect.New()
	default:
		return nil, fmt.Errorf("database: unsupported dialect")
	}

	if cfg.maxOpenConns > 0 {
		sqldb.SetMaxOpenConns(cfg.maxOpenConns)
	}
	if cfg.maxIdleConns > 0 {
		sqldb.SetMaxIdleConns(cfg.maxIdleConns)
	}
	if cfg.connMaxLifetime > 0 {
		sqldb.SetConnMaxLifetime(cfg.connMaxLifetime)
	}

	db := bun.NewDB(sqldb, dialect)

	if cfg.tracing {
		db.AddQueryHook(bunotel.NewQueryHook())
	}

	if !cfg.skipPing {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.pingTimeout)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("database: ping: %w", err)
		}
	}

	return db, nil
}
