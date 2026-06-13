// Copyright 2026 Rahmad Afandi. MIT License.

package parser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

const defaultKeysetLimit = 10

// KeysetColumn is one ordering column of a keyset. Name is the column (a plain
// or table-qualified identifier); Desc selects descending order. The columns,
// in order, must form a unique total ordering — make the last one a unique
// tiebreaker (typically the primary key).
type KeysetColumn struct {
	Name string
	Desc bool
}

// KeysetQuery holds keyset (cursor) paging parameters, bindable from the query
// string.
type KeysetQuery struct {
	Limit  int    `query:"limit"`
	Cursor string `query:"cursor"` // opaque; "" = first page
	Before bool   `query:"before"` // true = page backward (toward previous)
}

// Keyset returns a Bun query modifier that applies the keyset seek predicate,
// ORDER BY, and LIMIT+1 (one extra row, so pagination.NewCursorPage can detect
// a further page). The cursor is decoded into one value per column; an empty or
// malformed cursor yields the first page (no seek predicate). When any column
// name is not a simple identifier, ordering is skipped (defence-in-depth
// against injection) and only the limit is applied.
//
// Use it via query.Apply(Keyset(kq, cols)).
func Keyset(kq KeysetQuery, cols []KeysetColumn) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		limit := kq.Limit
		if limit <= 0 {
			limit = defaultKeysetLimit
		}

		if len(cols) == 0 || !keysetColumnsValid(cols) {
			return q.Limit(limit + 1)
		}

		if kq.Cursor != "" {
			if vals, err := DecodeCursor(kq.Cursor); err == nil && len(vals) == len(cols) {
				pred, args := buildSeek(cols, vals, kq.Before)
				q = q.Where(pred, args...)
			}
		}

		for _, c := range cols {
			dir := "ASC"
			if c.Desc != kq.Before { // effective direction, inverted when paging backward
				dir = "DESC"
			}
			q = q.OrderExpr(c.Name + " " + dir)
		}
		return q.Limit(limit + 1)
	}
}

// keysetColumnsValid reports whether every column name is a simple identifier.
func keysetColumnsValid(cols []KeysetColumn) bool {
	for _, c := range cols {
		if !isSimpleIdentifier(c.Name) {
			return false
		}
	}
	return true
}

// buildSeek builds the lexicographic seek predicate and its bound args for the
// given cursor values. For ORDER BY a ASC, b DESC it produces
// "((a > ?) OR (a = ? AND b < ?))" with args [av, av, bv]. The per-column
// comparison is ">" for an ascending effective direction and "<" for a
// descending one; effective direction is the column's Desc XOR before. Column
// names are assumed already validated by keysetColumnsValid.
func buildSeek(cols []KeysetColumn, vals []any, before bool) (string, []any) {
	groups := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols)*(len(cols)+1)/2)
	for i := range cols {
		parts := make([]string, 0, i+1)
		for j := 0; j < i; j++ {
			parts = append(parts, cols[j].Name+" = ?")
			args = append(args, vals[j])
		}
		op := ">"
		if cols[i].Desc != before {
			op = "<"
		}
		parts = append(parts, cols[i].Name+" "+op+" ?")
		args = append(args, vals[i])
		groups = append(groups, "("+strings.Join(parts, " AND ")+")")
	}
	return "(" + strings.Join(groups, " OR ") + ")", args
}

// EncodeCursor encodes column values into an opaque, URL-safe cursor (base64 of
// a JSON array).
func EncodeCursor(vals []any) string {
	b, _ := json.Marshal(vals)
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor reverses EncodeCursor. It returns an error for malformed input.
func DecodeCursor(s string) ([]any, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("parser: decode cursor: %w", err)
	}
	var vals []any
	if err := json.Unmarshal(b, &vals); err != nil {
		return nil, fmt.Errorf("parser: decode cursor: %w", err)
	}
	return vals, nil
}
