// Copyright 2026 Rahmad Afandi. MIT License.

package audit

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// bunSink inserts entries into a Bun database.
type bunSink struct {
	db *bun.DB
}

// NewBunSink returns a Sink that inserts entries into db.
func NewBunSink(db *bun.DB) Sink {
	return &bunSink{db: db}
}

func (s *bunSink) Record(ctx context.Context, e *Entry) error {
	if _, err := s.db.NewInsert().Model(e).Exec(ctx); err != nil {
		return fmt.Errorf("audit: insert entry: %w", err)
	}
	return nil
}

// Migrate creates the audit_log table (if absent) and an index on
// (actor, created_at).
func Migrate(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().Model((*Entry)(nil)).IfNotExists().Exec(ctx); err != nil {
		return fmt.Errorf("audit: create table: %w", err)
	}
	if _, err := db.NewCreateIndex().
		Model((*Entry)(nil)).
		Index("audit_log_actor_idx").
		Column("actor", "created_at").
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("audit: create index: %w", err)
	}
	return nil
}

// Filter narrows a List query; empty fields are ignored.
type Filter struct {
	Actor  string
	Action string
	Target string
	Limit  int
}

const defaultListLimit = 100

// List returns entries matching f, newest first.
func List(ctx context.Context, db *bun.DB, f Filter) ([]Entry, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	var entries []Entry
	q := db.NewSelect().Model(&entries)
	if f.Actor != "" {
		q = q.Where("actor = ?", f.Actor)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.Target != "" {
		q = q.Where("target = ?", f.Target)
	}
	if err := q.OrderExpr("id DESC").Limit(limit).Scan(ctx); err != nil {
		return nil, fmt.Errorf("audit: list entries: %w", err)
	}
	return entries, nil
}
