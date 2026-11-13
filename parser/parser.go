// Copyright 2026 Rahmad Afandi. MIT License.

// Package parser builds Bun pagination and search query modifiers from request query parameters.
package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
)

// MaxLimit caps the page size Paginate and Keyset will apply, so a client
// cannot request an unbounded page. Larger limits are clamped. Set it to 0 to
// disable the cap.
var MaxLimit = 1000

// clampLimit applies the default for a non-positive limit and the MaxLimit cap.
func clampLimit(limit, def int) int {
	if limit <= 0 {
		return def
	}
	if MaxLimit > 0 && limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

// PaginationQuery is a struct for holding pagination query parameters.
type PaginationQuery struct {
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Search string `query:"search"`
	Sort   string `query:"sort"`
	Order  string `query:"order"`
}

// Validate validates the pagination query parameters.
func (p *PaginationQuery) Validate(sortOptions []string) error {
	var errs []string
	if p.Page < 0 {
		errs = append(errs, "page must not be negative")
	}
	if p.Limit < 0 {
		errs = append(errs, "limit must not be negative")
	}
	if p.Sort != "" && !slices.Contains(sortOptions, p.Sort) {
		errs = append(errs, fmt.Sprintf("sort must be one of %s", strings.Join(sortOptions, ", ")))
	}
	if p.Order != "" && p.Order != "asc" && p.Order != "desc" {
		errs = append(errs, "order must be one of asc or desc")
	}
	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errs, ", "))
	}
	return nil
}

// Paginate returns a Bun query modifier that applies search, sorting, and
// page/limit offsets. Use it via query.Apply(Paginate(pq, columns)).
//
// pq.Page and pq.Limit are normalised in place to their defaults (1 and 10)
// when non-positive, and pq.Limit is clamped to MaxLimit. Search is
// case-insensitive: ILIKE on Postgres, LIKE elsewhere.
//
// Search matches anywhere in the column ("%term%"), which an ordinary index
// cannot serve, so its cost grows with the table. On Postgres, a pg_trgm GIN
// index on the searchable columns keeps it fast. Invalid sort/order values are ignored; call
// pq.Validate first to surface them as errors.
func Paginate(pq *PaginationQuery, columnsSearchable []string) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		q = applySearch(q, pq.Search, columnsSearchable)
		q = applySorting(q, pq.Sort, pq.Order)
		q = applyPaginationDefaults(q, pq)
		return q
	}
}

// Count returns a Bun query modifier that applies only the search filter,
// for use with a COUNT query. Use it via query.Apply(Count(search, columns)).
func Count(search string, columnsSearchable []string) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		return applySearch(q, search, columnsSearchable)
	}
}

func applyPaginationDefaults(q *bun.SelectQuery, pq *PaginationQuery) *bun.SelectQuery {
	if pq.Page <= 0 {
		pq.Page = 1
	}
	pq.Limit = clampLimit(pq.Limit, 10)
	offset := (pq.Page - 1) * pq.Limit
	return q.Offset(int64(offset)).Limit(int64(pq.Limit))
}

func applySearch(q *bun.SelectQuery, search string, columnsSearchable []string) *bun.SelectQuery {
	if search == "" || len(columnsSearchable) == 0 {
		return q
	}
	searchPattern := "%" + search + "%"
	// Only Postgres has ILIKE; LIKE is already case-insensitive (for ASCII) on
	// SQLite and under MySQL's default collations.
	op := " LIKE ?"
	if q.Dialect().Name() == dialect.PG {
		op = " ILIKE ?"
	}
	condition := strings.Join(columnsSearchable, op+" OR ") + op

	// Example:
	// columnsSearchable = []string{"name", "slug"}, search = "test"
	// WHERE name ILIKE '%test%' OR slug ILIKE '%test%'

	args := make([]any, len(columnsSearchable))
	for i := range columnsSearchable {
		args[i] = searchPattern
	}
	return q.Where(condition, args...)
}

func applySorting(q *bun.SelectQuery, sort, order string) *bun.SelectQuery {
	if sort == "" || order == "" {
		return q
	}
	order = strings.ToLower(order)
	if order != "asc" && order != "desc" {
		return q
	}
	if !isSimpleIdentifier(sort) {
		return q
	}
	return q.OrderExpr(sort + " " + order)
}

// isSimpleIdentifier reports whether s is a plain column name, optionally
// table-qualified (e.g. "name" or "a.created_at"), containing only letters,
// digits, underscores, and dots. This is defence-in-depth against SQL
// injection through OrderExpr; PaginationQuery.Validate remains the
// authoritative whitelist.
func isSimpleIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}
