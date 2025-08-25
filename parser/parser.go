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
	"gorm.io/gorm"
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
		errs = append(errs, "page must be greater than 0")
	}
	if p.Limit < 0 {
		errs = append(errs, "limit must be greater than 0")
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

func Paginate(pq *PaginationQuery, columnsSearchable []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		db = applySearch(db, pq.Search, columnsSearchable)
		db = applySorting(db, pq.Sort, pq.Order)
		db = applyPaginationDefaults(db, pq)

		return db
	}
}

func Count(search string, columnsSearchable []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		db = applySearch(db, search, columnsSearchable)
		return db
	}
}

func applyPaginationDefaults(db *gorm.DB, pq *PaginationQuery) *gorm.DB {
	if pq.Page <= 0 {
		pq.Page = 1
	}
	if pq.Limit <= 0 {
		pq.Limit = 10
	}
	offset := (pq.Page - 1) * pq.Limit
	return db.Offset(offset).Limit(pq.Limit)
}

func applySearch(db *gorm.DB, search string, columnsSearchable []string) *gorm.DB {
	if search == "" {
		return db
	}
	searchPattern := "%%" + search + "%%"
	searchCondition := strings.Join(columnsSearchable, " ILIKE ? OR ") + " ILIKE ?"

	// Example:
	// If columnsSearchable = []string{"name", "slug"} and search = "test"
	// Will generate SQL query:
	// WHERE name ILIKE '%%test%%' OR slug ILIKE '%%test%%'

	args := make([]interface{}, len(columnsSearchable))
	for i := range columnsSearchable {
		args[i] = searchPattern
	}
	return db.Where(searchCondition, args...)
}

func applySorting(db *gorm.DB, sort string, order string) *gorm.DB {
	if sort == "" || order == "" {
		return db
	}
	return db.Order(sort + " " + order)
}