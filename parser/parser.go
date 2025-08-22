package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
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
