// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
}

// Option configures the database connection.
type Option func(*config)

func WithMaxOpenConns(n int) Option              { return func(c *config) { c.maxOpenConns = n } }
func WithMaxIdleConns(n int) Option              { return func(c *config) { c.maxIdleConns = n } }
func WithConnMaxLifetime(d time.Duration) Option { return func(c *config) { c.connMaxLifetime = d } }
func WithPingTimeout(d time.Duration) Option     { return func(c *config) { c.pingTimeout = d } }

// WithoutPing skips the connect-time ping.
func WithoutPing() Option { return func(c *config) { c.skipPing = true } }

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
		if strings.Contains(dsn, "://") {
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
