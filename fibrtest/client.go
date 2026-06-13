// Copyright 2026 Rahmad Afandi. MIT License.

// Package fibrtest provides test helpers for fibr/Fiber apps: a fluent HTTP
// client over *fiber.App, response assertions, a JWT minting helper, and an
// in-memory Bun DB helper.
package fibrtest

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

const defaultTimeout = 10 * time.Second

// TB is the subset of *testing.T the harness needs, so callers pass a real
// *testing.T and tests can supply a stub. (testing.TB cannot be implemented
// outside the testing package.)
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Client drives a Fiber app under test.
type Client struct {
	t       TB
	app     *fiber.App
	headers http.Header
	timeout time.Duration
}

// New returns a Client for app.
func New(t TB, app *fiber.App) *Client {
	return &Client{
		t:       t,
		app:     app,
		headers: http.Header{},
		timeout: defaultTimeout,
	}
}

// clone returns a shallow copy of the Client with an independent header map.
func (c *Client) clone() *Client {
	h := make(http.Header, len(c.headers))
	for k, v := range c.headers {
		h[k] = append([]string(nil), v...)
	}
	return &Client{t: c.t, app: c.app, headers: h, timeout: c.timeout}
}

// WithBearer returns a copy of the Client that sends "Authorization: Bearer
// <token>" on every request.
func (c *Client) WithBearer(token string) *Client {
	nc := c.clone()
	nc.headers.Set("Authorization", "Bearer "+token)
	return nc
}

// WithHeader returns a copy of the Client that sends the given header on every
// request.
func (c *Client) WithHeader(key, value string) *Client {
	nc := c.clone()
	nc.headers.Set(key, value)
	return nc
}

// WithTimeout returns a copy of the Client using the given per-request timeout.
func (c *Client) WithTimeout(d time.Duration) *Client {
	nc := c.clone()
	nc.timeout = d
	return nc
}

// Request starts a builder for full control over a single request.
func (c *Client) Request(method, path string) *Request {
	return &Request{
		c:      c,
		method: method,
		path:   path,
		header: http.Header{},
	}
}

// Get sends a GET request.
func (c *Client) Get(path string) *Response {
	return c.Request(http.MethodGet, path).Do()
}

// Post sends a POST request; a non-nil body is JSON-encoded.
func (c *Client) Post(path string, body any) *Response {
	return c.bodyRequest(http.MethodPost, path, body)
}

// Put sends a PUT request; a non-nil body is JSON-encoded.
func (c *Client) Put(path string, body any) *Response {
	return c.bodyRequest(http.MethodPut, path, body)
}

// Patch sends a PATCH request; a non-nil body is JSON-encoded.
func (c *Client) Patch(path string, body any) *Response {
	return c.bodyRequest(http.MethodPatch, path, body)
}

// Delete sends a DELETE request.
func (c *Client) Delete(path string) *Response {
	return c.Request(http.MethodDelete, path).Do()
}

func (c *Client) bodyRequest(method, path string, body any) *Response {
	r := c.Request(method, path)
	if body != nil {
		r = r.JSON(body)
	}
	return r.Do()
}
