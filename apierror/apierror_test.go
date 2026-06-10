// Copyright 2026 Rahmad Afandi. MIT License.

package apierror_test

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/apierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructors(t *testing.T) {
	cases := []struct {
		err    *apierror.Error
		status int
		code   string
	}{
		{apierror.BadRequest("m"), 400, "bad_request"},
		{apierror.Unauthorized("m"), 401, "unauthorized"},
		{apierror.Forbidden("m"), 403, "forbidden"},
		{apierror.NotFound("m"), 404, "not_found"},
		{apierror.Conflict("m"), 409, "conflict"},
		{apierror.UnprocessableEntity("m"), 422, "unprocessable_entity"},
		{apierror.TooManyRequests("m"), 429, "too_many_requests"},
		{apierror.Internal("m"), 500, "internal_server_error"},
	}
	for _, c := range cases {
		assert.Equal(t, c.status, c.err.Status)
		assert.Equal(t, c.code, c.err.Code)
		assert.Equal(t, "m", c.err.Message)
	}
}

func TestBuilders(t *testing.T) {
	cause := errors.New("boom")
	e := apierror.Conflict("taken").WithCode("email_taken").WithDetails(map[string]int{"n": 1}).Wrap(cause)
	assert.Equal(t, "email_taken", e.Code)
	assert.Equal(t, map[string]int{"n": 1}, e.Details)
	assert.ErrorIs(t, e, cause)
	assert.Contains(t, e.Error(), "email_taken")
	assert.Contains(t, e.Error(), "taken")
}

func TestErrorsAs(t *testing.T) {
	var target *apierror.Error
	assert.False(t, errors.As(fiber.NewError(500, "x"), &target))
	assert.True(t, errors.As(apierror.NotFound("m"), &target))
}

func TestHandlerAPIError(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Get("/", func(c *fiber.Ctx) error {
		return apierror.NotFound("user not found").WithCode("user_not_found")
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 404, resp.StatusCode)
	assert.JSONEq(t, `{"code":404,"message":"user not found","error":"user_not_found","status":"error"}`, string(body))
}

func TestHandlerWithDetails(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Get("/", func(c *fiber.Ctx) error {
		return apierror.BadRequest("bad").WithDetails([]string{"x"})
	})
	resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 400, resp.StatusCode)
	assert.JSONEq(t, `{"code":400,"message":"bad","error":"bad_request","data":["x"],"status":"error"}`, string(body))
}

func TestHandlerWrappedCauseNoLeak(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Get("/", func(c *fiber.Ctx) error {
		return apierror.Internal("oops").Wrap(errors.New("db password leaked"))
	})
	resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 500, resp.StatusCode)
	assert.NotContains(t, string(body), "db password leaked")
}

func TestHandlerFiberError(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Get("/", func(c *fiber.Ctx) error { return fiber.NewError(403, "nope") })
	resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 403, resp.StatusCode)
	assert.Contains(t, string(body), `"status":"error"`)
	assert.Contains(t, string(body), "nope")
}

func TestHandlerGenericNoLeak(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Get("/", func(c *fiber.Ctx) error { return errors.New("secret internal detail") })
	resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 500, resp.StatusCode)
	assert.NotContains(t, string(body), "secret internal detail")
	assert.Contains(t, string(body), `"status":"error"`)
}
