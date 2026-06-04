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

package parser

import (
	"bytes"
	"context"
	"database/sql"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestParser(t *testing.T) {
	app := fiber.New()

	type TestStruct struct {
		Name string `json:"name"`
	}

	t.Run("ParseBody", func(t *testing.T) {
		app.Post("/body", func(c *fiber.Ctx) error {
			body, err := ParseBody[TestStruct](c)
			assert.NoError(t, err)
			assert.Equal(t, "test", body.Name)
			return nil
		})

		req := httptest.NewRequest("POST", "/body", bytes.NewBuffer([]byte(`{"name":"test"}`)))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	})

	t.Run("ParseQuery", func(t *testing.T) {
		app.Get("/query", func(c *fiber.Ctx) error {
			query, err := ParseQuery[TestStruct](c)
			assert.NoError(t, err)
			assert.Equal(t, "test", query.Name)
			return nil
		})

		req := httptest.NewRequest("GET", "/query?name=test", nil)
		app.Test(req)
	})

	t.Run("ParseParams", func(t *testing.T) {
		app.Get("/params/:name", func(c *fiber.Ctx) error {
			params, err := ParseParams[TestStruct](c)
			assert.NoError(t, err)
			assert.Equal(t, "test", params.Name)
			return nil
		})

		req := httptest.NewRequest("GET", "/params/test", nil)
		app.Test(req)
	})
}

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
