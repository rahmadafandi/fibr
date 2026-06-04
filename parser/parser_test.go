// Copyright 2026 Rahmad Afandi. MIT License.

package parser

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestPaginationQueryValidate(t *testing.T) {
	t.Run("ZeroIsAllowed", func(t *testing.T) {
		pq := &PaginationQuery{Page: 0, Limit: 0}
		assert.NoError(t, pq.Validate(nil))
	})

	t.Run("NegativeRejected", func(t *testing.T) {
		pq := &PaginationQuery{Page: -1, Limit: -1}
		err := pq.Validate(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "page")
		assert.Contains(t, err.Error(), "limit")
	})

	t.Run("InvalidOrder", func(t *testing.T) {
		pq := &PaginationQuery{Order: "sideways"}
		err := pq.Validate(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "order")
	})

	t.Run("InvalidSort", func(t *testing.T) {
		pq := &PaginationQuery{Sort: "evil"}
		err := pq.Validate([]string{"name", "created_at"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sort")
	})
}

type article struct {
	bun.BaseModel `bun:"table:articles,alias:a"`
	ID            int64  `bun:"id,pk,autoincrement"`
	Name          string `bun:"name"`
}

func newBunDB(t *testing.T) *bun.DB {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	assert.NoError(t, err)
	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	_, err = db.NewCreateTable().Model((*article)(nil)).Exec(ctx)
	assert.NoError(t, err)
	for _, name := range []string{"alpha", "bravo", "charlie", "delta", "echo"} {
		_, err = db.NewInsert().Model(&article{Name: name}).Exec(ctx)
		assert.NoError(t, err)
	}
	return db
}

func TestPaginateLimitOffset(t *testing.T) {
	db := newBunDB(t)
	ctx := context.Background()

	pq := &PaginationQuery{Page: 1, Limit: 2}
	var got []article
	err := db.NewSelect().Model(&got).Apply(Paginate(pq, nil)).Scan(ctx)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestPaginateDefaults(t *testing.T) {
	db := newBunDB(t)
	ctx := context.Background()

	pq := &PaginationQuery{}
	var got []article
	err := db.NewSelect().Model(&got).Apply(Paginate(pq, nil)).Scan(ctx)
	assert.NoError(t, err)
	assert.Len(t, got, 5)
}

func TestPaginateSorting(t *testing.T) {
	db := newBunDB(t)
	ctx := context.Background()

	pq := &PaginationQuery{Sort: "name", Order: "desc", Limit: 1}
	var got []article
	err := db.NewSelect().Model(&got).Apply(Paginate(pq, nil)).Scan(ctx)
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "echo", got[0].Name)
}

func TestApplySearchBuildsILIKECondition(t *testing.T) {
	// SQLite has no ILIKE, so we don't execute the search; we verify the
	// generated SQL contains the ILIKE condition.
	db := newBunDB(t)
	q := db.NewSelect().Model((*article)(nil)).Apply(Count("foo", []string{"name", "a.id"}))
	sqlStr := q.String()
	assert.Contains(t, sqlStr, "ILIKE")
	assert.Contains(t, sqlStr, "name")
}

func TestApplySortingRejectsInjection(t *testing.T) {
	db := newBunDB(t)
	// A malicious sort value must not appear in the generated SQL.
	pq := &PaginationQuery{Sort: "name; DROP TABLE articles", Order: "asc", Limit: 5}
	q := db.NewSelect().Model((*article)(nil)).Apply(Paginate(pq, nil))
	sqlStr := q.String()
	assert.NotContains(t, sqlStr, "DROP TABLE")

	// A valid identifier IS applied.
	pq2 := &PaginationQuery{Sort: "name", Order: "asc", Limit: 5}
	q2 := db.NewSelect().Model((*article)(nil)).Apply(Paginate(pq2, nil))
	assert.Contains(t, q2.String(), "ORDER BY")
}
