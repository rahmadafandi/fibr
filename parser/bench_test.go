// Copyright 2026 Rahmad Afandi. MIT License.

package parser

import (
	"context"
	"fmt"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/rahmadafandi/fibr/v2/fibrtest"
)

// BenchmarkPaginateBuild is the cost of building the query (search + sort +
// page) without touching a database.
func BenchmarkPaginateBuild(b *testing.B) {
	db := bun.NewDB(nil, pgdialect.New())
	cols := []string{"name", "a.id"}
	b.ReportAllocs()
	for b.Loop() {
		pq := &PaginationQuery{Page: 3, Limit: 20, Search: "alpha", Sort: "name", Order: "asc"}
		_ = db.NewSelect().Model((*article)(nil)).Apply(Paginate(pq, cols)).String()
	}
}

func benchArticles(b *testing.B, rows int) *bun.DB {
	b.Helper()
	db := fibrtest.NewDB(b)
	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*article)(nil)).Exec(ctx); err != nil {
		b.Fatal(err)
	}
	batch := make([]article, rows)
	for i := range batch {
		batch[i].Name = fmt.Sprintf("article number %06d", i)
	}
	if _, err := db.NewInsert().Model(&batch).Exec(ctx); err != nil {
		b.Fatal(err)
	}
	return db
}

// BenchmarkSearchPage runs a searched page against SQLite at two table sizes.
// The search is a "%term%" match, which no ordinary index can serve, so the
// cost grows with the table. (SQLite stands in for Postgres here; the shape of
// the growth is the point, not the absolute numbers.)
func BenchmarkSearchPage(b *testing.B) {
	for _, rows := range []int{1_000, 20_000} {
		b.Run(fmt.Sprintf("rows=%d", rows), func(b *testing.B) {
			db := benchArticles(b, rows)
			ctx := context.Background()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				var got []article
				// A term that matches nothing forces a full scan.
				pq := &PaginationQuery{Page: 1, Limit: 20, Search: "zzz"}
				if err := db.NewSelect().Model(&got).Apply(Paginate(pq, []string{"name"})).Scan(ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
