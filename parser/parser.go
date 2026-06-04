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
	"fmt"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

// ParseBody parses the request body into the provided generic type T.
func ParseBody[T any](c *fiber.Ctx) (*T, error) {
	var body T
	if err := c.BodyParser(&body); err != nil {
		return nil, err
	}
	return &body, nil
}

// ParseParams parses the route parameters into the provided generic type T.
func ParseParams[T any](c *fiber.Ctx) (*T, error) {
	var params T
	if err := c.ParamsParser(&params); err != nil {
		return nil, err
	}
	return &params, nil
}

// ParseQuery parses the query parameters into the provided generic type T.
func ParseQuery[T any](c *fiber.Ctx) (*T, error) {
	var query T
	if err := c.QueryParser(&query); err != nil {
		return nil, err
	}
	return &query, nil
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
	if pq.Limit <= 0 {
		pq.Limit = 10
	}
	offset := (pq.Page - 1) * pq.Limit
	return q.Offset(offset).Limit(pq.Limit)
}

func applySearch(q *bun.SelectQuery, search string, columnsSearchable []string) *bun.SelectQuery {
	if search == "" || len(columnsSearchable) == 0 {
		return q
	}
	searchPattern := "%" + search + "%"
	condition := strings.Join(columnsSearchable, " ILIKE ? OR ") + " ILIKE ?"

	// Example:
	// columnsSearchable = []string{"name", "slug"}, search = "test"
	// WHERE name ILIKE '%test%' OR slug ILIKE '%test%'

	args := make([]interface{}, len(columnsSearchable))
	for i := range columnsSearchable {
		args[i] = searchPattern
	}
	return q.Where(condition, args...)
}

func applySorting(q *bun.SelectQuery, sort, order string) *bun.SelectQuery {
	if sort == "" || order == "" {
		return q
	}
	return q.OrderExpr(sort + " " + order)
}
