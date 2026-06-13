// Copyright 2026 Rahmad Afandi. MIT License.

package pagination_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/rahmadafandi/fibr/pagination"
	"github.com/rahmadafandi/fibr/parser"
)

type row struct {
	bun.BaseModel `bun:"table:rows,alias:r"`
	ID            int64  `bun:"id,pk,autoincrement"`
	Name          string `bun:"name"`
}

var keysetCols = []parser.KeysetColumn{{Name: "name"}, {Name: "id"}}

func extractRow(r row) []any { return []any{r.Name, r.ID} }

func newDB(t *testing.T, names ...string) (*bun.DB, context.Context) {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	_, err = db.NewCreateTable().Model((*row)(nil)).Exec(ctx)
	require.NoError(t, err)
	for _, n := range names {
		_, err = db.NewInsert().Model(&row{Name: n}).Exec(ctx)
		require.NoError(t, err)
	}
	return db, ctx
}

func fetch(t *testing.T, db *bun.DB, ctx context.Context, kq parser.KeysetQuery) *pagination.CursorPage[row] {
	t.Helper()
	var rows []row
	require.NoError(t, db.NewSelect().Model(&rows).Apply(parser.Keyset(kq, keysetCols)).Scan(ctx))
	return pagination.NewCursorPage(rows, kq, keysetCols, extractRow)
}

func names(p *pagination.CursorPage[row]) []string {
	out := make([]string, len(p.Data))
	for i, r := range p.Data {
		out[i] = r.Name
	}
	return out
}

func TestCursorForwardFullSweep(t *testing.T) {
	db, ctx := newDB(t, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j")

	var seen []string
	kq := parser.KeysetQuery{Limit: 3}
	for {
		p := fetch(t, db, ctx, kq)
		seen = append(seen, names(p)...)
		if p.NextCursor == "" {
			break
		}
		kq = parser.KeysetQuery{Limit: 3, Cursor: p.NextCursor}
	}
	assert.Equal(t, []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, seen)
}

func TestCursorFirstPageNoPrevLastPageNoNext(t *testing.T) {
	db, ctx := newDB(t, "a", "b", "c", "d", "e")

	first := fetch(t, db, ctx, parser.KeysetQuery{Limit: 2})
	assert.Equal(t, []string{"a", "b"}, names(first))
	assert.Empty(t, first.PrevCursor)
	assert.NotEmpty(t, first.NextCursor)

	// Walk to the last page.
	p := first
	for p.NextCursor != "" {
		p = fetch(t, db, ctx, parser.KeysetQuery{Limit: 2, Cursor: p.NextCursor})
	}
	assert.Equal(t, []string{"e"}, names(p))
	assert.Empty(t, p.NextCursor)
}

func TestCursorBackwardReturnsPreviousPage(t *testing.T) {
	db, ctx := newDB(t, "a", "b", "c", "d", "e", "f")

	// Forward to page 2: {c, d}.
	page1 := fetch(t, db, ctx, parser.KeysetQuery{Limit: 2})
	page2 := fetch(t, db, ctx, parser.KeysetQuery{Limit: 2, Cursor: page1.NextCursor})
	require.Equal(t, []string{"c", "d"}, names(page2))
	require.NotEmpty(t, page2.PrevCursor)

	// Backward from page 2 -> page 1: {a, b}, in display order.
	back := fetch(t, db, ctx, parser.KeysetQuery{Limit: 2, Cursor: page2.PrevCursor, Before: true})
	assert.Equal(t, []string{"a", "b"}, names(back))
}

func TestCursorTieBreakByID(t *testing.T) {
	// Three rows share the name "same"; the id tiebreaker must order them with
	// no duplicates or skips across page boundaries.
	db, ctx := newDB(t, "same", "same", "same", "zzz")

	var ids []int64
	kq := parser.KeysetQuery{Limit: 2}
	for {
		var rows []row
		require.NoError(t, db.NewSelect().Model(&rows).Apply(parser.Keyset(kq, keysetCols)).Scan(ctx))
		p := pagination.NewCursorPage(rows, kq, keysetCols, extractRow)
		for _, r := range p.Data {
			ids = append(ids, r.ID)
		}
		if p.NextCursor == "" {
			break
		}
		kq = parser.KeysetQuery{Limit: 2, Cursor: p.NextCursor}
	}
	assert.Equal(t, []int64{1, 2, 3, 4}, ids)
}

func TestCursorEmptyTable(t *testing.T) {
	db, ctx := newDB(t)
	p := fetch(t, db, ctx, parser.KeysetQuery{Limit: 2})
	assert.Empty(t, p.Data)
	assert.Empty(t, p.NextCursor)
	assert.Empty(t, p.PrevCursor)
}
