// Copyright 2026 Rahmad Afandi. MIT License.

package parser

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/rahmadafandi/fibr/v2/fibrtest"
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
	db := fibrtest.NewDB(t)

	ctx := context.Background()
	_, err := db.NewCreateTable().Model((*article)(nil)).Exec(ctx)
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

func TestSearchRunsOnSQLiteCaseInsensitively(t *testing.T) {
	db := newBunDB(t)
	var got []article
	err := db.NewSelect().Model(&got).Apply(Count("ALPH", []string{"name"})).Scan(context.Background())
	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestSearchUsesILIKEOnPostgres(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())
	sqlStr := db.NewSelect().Model((*article)(nil)).Apply(Count("foo", []string{"name", "a.id"})).String()
	assert.Contains(t, sqlStr, "name ILIKE '%foo%' OR a.id ILIKE '%foo%'")
}

func TestLimitClampedToMaxLimit(t *testing.T) {
	db := newBunDB(t)

	pq := &PaginationQuery{Limit: MaxLimit + 1}
	q := db.NewSelect().Model((*article)(nil)).Apply(Paginate(pq, nil))
	assert.Equal(t, MaxLimit, pq.Limit)
	assert.Contains(t, q.String(), fmt.Sprintf("LIMIT %d", MaxLimit))

	kq := db.NewSelect().Model((*article)(nil)).Apply(Keyset(KeysetQuery{Limit: MaxLimit * 10}, nil))
	assert.Contains(t, kq.String(), fmt.Sprintf("LIMIT %d", MaxLimit+1))
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
