// Copyright 2026 Rahmad Afandi. MIT License.

// Package apierror provides typed HTTP errors and a Fiber ErrorHandler that
// renders them (and *fiber.Error, and unknown errors) as the standard JSON
// envelope. Handlers and services return an *Error; bootstrap wires Handler by
// default so the response is consistent.
package apierror

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/response"
)

// Error is a typed HTTP error: an HTTP status, a machine-readable code, a human
// message, optional details (rendered into "data"), and an optional wrapped
// cause (for errors.Is/As and logging; never serialized).
type Error struct {
	Status  int
	Code    string
	Message string
	Details any
	err     error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped cause, if any.
func (e *Error) Unwrap() error { return e.err }

// WithCode overrides the default machine code (e.g. "email_taken").
func (e *Error) WithCode(code string) *Error { e.Code = code; return e }

// WithDetails attaches details rendered into the response "data" field.
func (e *Error) WithDetails(d any) *Error { e.Details = d; return e }

// Wrap attaches an underlying cause. The cause is never exposed in the response.
func (e *Error) Wrap(err error) *Error { e.err = err; return e }

// New builds an Error with an explicit status and code.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// BadRequest returns a 400 Error.
func BadRequest(message string) *Error { return New(fiber.StatusBadRequest, "bad_request", message) }

// Unauthorized returns a 401 Error.
func Unauthorized(message string) *Error {
	return New(fiber.StatusUnauthorized, "unauthorized", message)
}

// Forbidden returns a 403 Error.
func Forbidden(message string) *Error { return New(fiber.StatusForbidden, "forbidden", message) }

// NotFound returns a 404 Error.
func NotFound(message string) *Error { return New(fiber.StatusNotFound, "not_found", message) }

// Conflict returns a 409 Error.
func Conflict(message string) *Error { return New(fiber.StatusConflict, "conflict", message) }

// UnprocessableEntity returns a 422 Error.
func UnprocessableEntity(message string) *Error {
	return New(fiber.StatusUnprocessableEntity, "unprocessable_entity", message)
}

// TooManyRequests returns a 429 Error.
func TooManyRequests(message string) *Error {
	return New(fiber.StatusTooManyRequests, "too_many_requests", message)
}

// Internal returns a 500 Error.
func Internal(message string) *Error {
	return New(fiber.StatusInternalServerError, "internal", message)
}

// Handler is a fiber.ErrorHandler that renders errors as the standard JSON
// envelope: *Error at its status with its code, *fiber.Error at its code, and
// any other error as a generic 500 (the raw message is not exposed).
func Handler(c *fiber.Ctx, err error) error {
	var ae *Error
	if errors.As(err, &ae) {
		return c.Status(ae.Status).JSON(&response.Response{
			Code:    ae.Status,
			Message: ae.Message,
			Error:   ae.Code,
			Data:    ae.Details,
			Status:  "error",
		})
	}

	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(&response.Response{
			Code:    fe.Code,
			Message: fe.Message,
			Status:  "error",
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(&response.Response{
		Code:    fiber.StatusInternalServerError,
		Message: "internal server error",
		Status:  "error",
	})
}
