// Copyright 2026 Rahmad Afandi. MIT License.

// Package migrate provides thin, testable helpers over bun/migrate plus a ready
// cobra command for running migrations from an application binary.
package migrate

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	bunmigrate "github.com/uptrace/bun/migrate"
)

// Migrations is the bun migration collection. Re-exported so projects can
// declare migrations without importing bun/migrate directly.
type Migrations = bunmigrate.Migrations

// MigrationsOption configures a Migrations collection.
type MigrationsOption = bunmigrate.MigrationsOption

// NewMigrations creates a migration collection.
func NewMigrations(opts ...MigrationsOption) *Migrations {
	return bunmigrate.NewMigrations(opts...)
}

// WithMigrationsDirectory pins the directory used by Create (so it does not rely
// on runtime.Caller).
var WithMigrationsDirectory = bunmigrate.WithMigrationsDirectory

func newMigrator(ctx context.Context, db *bun.DB, ms *Migrations) (*bunmigrate.Migrator, error) {
	m := bunmigrate.NewMigrator(db, ms)
	if err := m.Init(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

// Up applies all pending migrations and returns a human-readable summary.
func Up(ctx context.Context, db *bun.DB, ms *Migrations) (string, error) {
	m, err := newMigrator(ctx, db, ms)
	if err != nil {
		return "", err
	}
	if err := m.Lock(ctx); err != nil {
		return "", err
	}
	defer m.Unlock(ctx) //nolint:errcheck

	group, err := m.Migrate(ctx)
	if err != nil {
		return "", err
	}
	if group.IsZero() {
		return "no new migrations to run", nil
	}
	return fmt.Sprintf("migrated: %s", group), nil
}

// Down rolls back the last applied migration group.
func Down(ctx context.Context, db *bun.DB, ms *Migrations) (string, error) {
	m, err := newMigrator(ctx, db, ms)
	if err != nil {
		return "", err
	}
	if err := m.Lock(ctx); err != nil {
		return "", err
	}
	defer m.Unlock(ctx) //nolint:errcheck

	group, err := m.Rollback(ctx)
	if err != nil {
		return "", err
	}
	if group.IsZero() {
		return "nothing to roll back", nil
	}
	return fmt.Sprintf("rolled back: %s", group), nil
}

// Status returns applied and pending migrations.
func Status(ctx context.Context, db *bun.DB, ms *Migrations) (string, error) {
	m, err := newMigrator(ctx, db, ms)
	if err != nil {
		return "", err
	}
	// No lock needed: MigrationsWithStatus is a read-only query.
	all, err := m.MigrationsWithStatus(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("applied: %s\npending: %s", all.Applied(), all.Unapplied()), nil
}

// Create scaffolds a new empty Go migration file in the migrations directory
// (which must already exist). The generated file uses package "migrations";
// keep the migrations directory named "migrations" (the generated layout uses
// internal/migrations). It does not need a database connection.
func Create(ms *Migrations, name string) (string, error) {
	m := bunmigrate.NewMigrator(nil, ms)
	mf, err := m.CreateGoMigration(context.Background(), name)
	if err != nil {
		return "", err
	}
	return mf.Path, nil
}
