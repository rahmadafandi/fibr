// Copyright 2026 Rahmad Afandi. MIT License.

// Package http is a context-aware JSON HTTP client with retries.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"sync"
	"time"

	"github.com/rahmadafandi/fibr/logger"
	"github.com/valyala/fasthttp"
)

// HTTP method constants re-exported from fasthttp for use with HTTP.FireAndForget
// and similar helpers.
const (
	Post    = fasthttp.MethodPost
	Get     = fasthttp.MethodGet
	Put     = fasthttp.MethodPut
	Patch   = fasthttp.MethodPatch
	Delete  = fasthttp.MethodDelete
	Options = fasthttp.MethodOptions
	Head    = fasthttp.MethodHead
	Connect = fasthttp.MethodConnect
	Trace   = fasthttp.MethodTrace
)

// Error is returned when the server responds with a non-2xx status code.
type Error struct {
	Code int
	Body []byte
}

func (e *Error) Error() string {
	return fmt.Sprintf("http: status %d: %s", e.Code, string(e.Body))
}

// HTTP is a small JSON HTTP client built on fasthttp.
type HTTP struct {
	BaseURL string
	Client  *fasthttp.Client

	mu         sync.RWMutex
	headers    map[string]string
	timeout    time.Duration
	retries    int
	backoff    time.Duration
	breaker    *breaker
	ctxHeaders []ctxHeaderRule
	logger     *logger.Logger
}

// ctxHeaderRule derives a request header value from the call's context.
type ctxHeaderRule struct {
	name string
	fn   func(context.Context) string
}

// Option configures an HTTP client.
type Option func(*HTTP)

// WithTimeout sets the per-request timeout used when no context deadline is present.
func WithTimeout(d time.Duration) Option { return func(h *HTTP) { h.timeout = d } }

// WithRetry configures the number of additional retry attempts and the backoff
// duration between them. Only 5xx errors and transport failures are retried.
func WithRetry(n int, backoff time.Duration) Option {
	return func(h *HTTP) { h.retries = n; h.backoff = backoff }
}

// WithCircuitBreaker enables a circuit breaker: after maxFailures consecutive
// failures (transport errors or 5xx responses) the client rejects requests with
// ErrCircuitOpen until openTimeout elapses, then allows a single probe.
func WithCircuitBreaker(maxFailures int, openTimeout time.Duration) Option {
	return func(h *HTTP) { h.breaker = newBreaker(maxFailures, openTimeout) }
}

// WithHeader adds a default header sent with every request.
func WithHeader(key, value string) Option { return func(h *HTTP) { h.headers[key] = value } }

// WithContextHeader sets a per-request header from the call's context: before
// each request, extract is called with the context and, when it returns a
// non-empty string, that value is sent as header. Useful for propagating a
// request id, trace correlation, or tenant downstream.
func WithContextHeader(header string, extract func(context.Context) string) Option {
	return func(h *HTTP) {
		h.ctxHeaders = append(h.ctxHeaders, ctxHeaderRule{name: header, fn: extract})
	}
}

// WithClient replaces the underlying fasthttp client.
func WithClient(c *fasthttp.Client) Option { return func(h *HTTP) { h.Client = c } }

// WithLogger attaches a logger used to report FireAndForget errors.
func WithLogger(l *logger.Logger) Option { return func(h *HTTP) { h.logger = l } }

// New creates a new HTTP client.
func New(baseURL string, opts ...Option) *HTTP {
	h := &HTTP{
		BaseURL: baseURL,
		Client:  &fasthttp.Client{},
		headers: make(map[string]string),
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

// SetHeader sets a default header sent with every request.
func (h *HTTP) SetHeader(key, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.headers[key] = value
}

func (h *HTTP) snapshotHeaders() map[string]string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[string]string, len(h.headers))
	for k, v := range h.headers {
		out[k] = v
	}
	return out
}

// Get sends a GET request to path and JSON-decodes the response into out.
func (h *HTTP) Get(ctx context.Context, path string, out any) (int, error) {
	return h.request(ctx, fasthttp.MethodGet, path, nil, out)
}

// Post sends a POST request with a JSON-encoded body and decodes the response into out.
func (h *HTTP) Post(ctx context.Context, path string, body, out any) (int, error) {
	return h.request(ctx, fasthttp.MethodPost, path, body, out)
}

// Put sends a PUT request with a JSON-encoded body and decodes the response into out.
func (h *HTTP) Put(ctx context.Context, path string, body, out any) (int, error) {
	return h.request(ctx, fasthttp.MethodPut, path, body, out)
}

// Patch sends a PATCH request with a JSON-encoded body and decodes the response into out.
func (h *HTTP) Patch(ctx context.Context, path string, body, out any) (int, error) {
	return h.request(ctx, fasthttp.MethodPatch, path, body, out)
}

// Delete sends a DELETE request to path and JSON-decodes the response into out.
func (h *HTTP) Delete(ctx context.Context, path string, out any) (int, error) {
	return h.request(ctx, fasthttp.MethodDelete, path, nil, out)
}

// FileField is one file part of a multipart request.
type FileField struct {
	Field    string    // form field name
	Filename string    // file name reported to the server
	Content  io.Reader // file contents
}

// PostForm sends an application/x-www-form-urlencoded POST and JSON-decodes the
// response into out.
func (h *HTTP) PostForm(ctx context.Context, path string, values url.Values, out any) (int, error) {
	return h.requestRaw(ctx, fasthttp.MethodPost, path, []byte(values.Encode()),
		"application/x-www-form-urlencoded", out)
}

// PostMultipart sends a multipart/form-data POST with the given fields and files
// and JSON-decodes the response into out.
func (h *HTTP) PostMultipart(ctx context.Context, path string, fields map[string]string, files []FileField, out any) (int, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return 0, err
		}
	}
	for _, f := range files {
		fw, err := w.CreateFormFile(f.Field, f.Filename)
		if err != nil {
			return 0, err
		}
		if _, err := io.Copy(fw, f.Content); err != nil {
			return 0, err
		}
	}
	if err := w.Close(); err != nil {
		return 0, err
	}
	return h.requestRaw(ctx, fasthttp.MethodPost, path, buf.Bytes(), w.FormDataContentType(), out)
}

// FireAndForget sends a request in the background, logging any error if a
// logger was configured via WithLogger. The caller's context values are kept
// but its cancellation/deadline are dropped so the request outlives the caller.
func (h *HTTP) FireAndForget(ctx context.Context, method, path string, body any) {
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		if _, err := h.request(bgCtx, method, path, body, nil); err != nil && h.logger != nil {
			h.logger.Error(err, "fire and forget request failed", "method", method, "path", path)
		}
	}()
}

func (h *HTTP) request(ctx context.Context, method, path string, body, out any) (int, error) {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		payload = b
	}
	return h.requestRaw(ctx, method, path, payload, "application/json", out)
}

// requestRaw sends a pre-encoded body with an explicit content type, applying
// the configured retry/backoff policy and JSON-decoding the response into out.
func (h *HTTP) requestRaw(ctx context.Context, method, path string, payload []byte, contentType string, out any) (int, error) {
	if h.breaker != nil {
		if !h.breaker.allow() {
			return 0, ErrCircuitOpen
		}
		code, err := h.requestRawAttempts(ctx, method, path, payload, contentType, out)
		if breakerFailure(code, err) {
			h.breaker.onFailure()
		} else {
			h.breaker.onSuccess()
		}
		return code, err
	}
	return h.requestRawAttempts(ctx, method, path, payload, contentType, out)
}

// breakerFailure reports whether an outcome counts as a dependency failure for
// the circuit breaker: a transport error (no status) or a 5xx response. Client
// errors (4xx) and decode errors on a 2xx do not trip the breaker.
func breakerFailure(code int, err error) bool {
	if err != nil && code == 0 {
		return true
	}
	return code >= 500
}

// requestRawAttempts runs the retry/backoff loop for a single logical request.
func (h *HTTP) requestRawAttempts(ctx context.Context, method, path string, payload []byte, contentType string, out any) (int, error) {
	headers := h.snapshotHeaders()
	for _, r := range h.ctxHeaders {
		if v := r.fn(ctx); v != "" {
			headers[r.name] = v
		}
	}
	attempts := h.retries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	lastCode := 0

	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}

		code, respBody, err := h.doOnce(ctx, method, path, payload, contentType, headers)
		if err != nil {
			lastErr = err
			lastCode = 0
			if i < attempts-1 {
				if serr := sleepWithContext(ctx, h.backoff); serr != nil {
					return 0, serr
				}
			}
			continue
		}

		if code < 200 || code >= 300 {
			httpErr := &Error{Code: code, Body: respBody}
			// Retry only server errors (5xx). Client errors (4xx) are returned immediately.
			if code >= 500 && i < attempts-1 {
				lastErr = httpErr
				lastCode = code
				if serr := sleepWithContext(ctx, h.backoff); serr != nil {
					return 0, serr
				}
				continue
			}
			return code, httpErr
		}

		if out != nil {
			if err := json.Unmarshal(respBody, out); err != nil {
				return code, err
			}
		}
		return code, nil
	}

	if httpErr, ok := lastErr.(*Error); ok {
		return httpErr.Code, httpErr
	}
	return lastCode, lastErr
}

// sleepWithContext waits for d or until ctx is done, whichever comes first.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *HTTP) doOnce(ctx context.Context, method, path string, payload []byte, contentType string, headers map[string]string) (int, []byte, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(h.BaseURL + path)
	req.Header.SetMethod(method)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if payload != nil {
		req.SetBody(payload)
		if contentType == "" {
			contentType = "application/json"
		}
		req.Header.SetContentType(contentType)
	}

	var err error
	if dl, ok := ctx.Deadline(); ok {
		err = h.Client.DoDeadline(req, resp, dl)
	} else if h.timeout > 0 {
		err = h.Client.DoTimeout(req, resp, h.timeout)
	} else {
		err = h.Client.Do(req, resp)
	}
	if err != nil {
		return 0, nil, err
	}

	bodyCopy := append([]byte(nil), resp.Body()...)
	return resp.StatusCode(), bodyCopy, nil
}
