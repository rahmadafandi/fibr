// Copyright 2026 Rahmad Afandi. MIT License.

// Package bind parses and validates request input in one call. Each function
// decodes the request into T, runs validator.ValidateStruct, and on failure
// writes a structured error response — 400 for malformed input, 422 for
// validation errors — returning ok=false so the handler can stop with
// `return nil`.
package bind

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/response"
	"github.com/rahmadafandi/fibr/validator"
)

// Body decodes the request body (JSON/form/multipart, by Content-Type) into T
// and validates it. On success it returns (value, true). On failure it writes
// the error response and returns (zero, false).
func Body[T any](c *fiber.Ctx) (T, bool) { return bindAndValidate[T](c, c.BodyParser) }

// Query decodes the URL query string into T and validates it. See Body.
func Query[T any](c *fiber.Ctx) (T, bool) { return bindAndValidate[T](c, c.QueryParser) }

// Params decodes the route path parameters into T and validates it. See Body.
func Params[T any](c *fiber.Ctx) (T, bool) { return bindAndValidate[T](c, c.ParamsParser) }

func bindAndValidate[T any](c *fiber.Ctx, parse func(any) error) (T, bool) {
	var v T
	if err := parse(&v); err != nil {
		_ = response.SendError(c, nil, err.Error(), fiber.StatusBadRequest)
		return v, false
	}
	if errs := validator.ValidateStruct(v); len(errs) > 0 {
		_ = response.SendError(c, errs, "validation failed", fiber.StatusUnprocessableEntity)
		return v, false
	}
	return v, true
}
