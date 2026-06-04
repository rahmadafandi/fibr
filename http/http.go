// Copyright 2026 Rahmad Afandi. MIT License.

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rahmadafandi/fiber-helpers/logger"
	"github.com/valyala/fasthttp"
)

// HTTP method constants re-exported from fasthttp for use with Http.FireAndForget
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

// HTTPError is returned when the server responds with a non-2xx status code.
type HTTPError struct {
	Code int
	Body []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http: status %d: %s", e.Code, string(e.Body))
}

// Http is a small JSON HTTP client built on fasthttp.
type Http struct {
	BaseURL string
	Client  *fasthttp.Client

	mu      sync.RWMutex
	headers map[string]string
	timeout time.Duration
	retries int
	backoff time.Duration
	logger  *logger.Logger
}

// Option configures an Http client.
type Option func(*Http)

// WithTimeout sets the per-request timeout used when no context deadline is present.
func WithTimeout(d time.Duration) Option { return func(h *Http) { h.timeout = d } }

// WithRetry configures the number of additional retry attempts and the backoff
// duration between them. Only 5xx errors and transport failures are retried.
func WithRetry(n int, backoff time.Duration) Option {
	return func(h *Http) { h.retries = n; h.backoff = backoff }
}

// WithHeader adds a default header sent with every request.
func WithHeader(key, value string) Option { return func(h *Http) { h.headers[key] = value } }

// WithClient replaces the underlying fasthttp client.
func WithClient(c *fasthttp.Client) Option { return func(h *Http) { h.Client = c } }

// WithLogger attaches a logger used to report FireAndForget errors.
func WithLogger(l *logger.Logger) Option { return func(h *Http) { h.logger = l } }

// New creates a new Http client.
func New(baseURL string, opts ...Option) *Http {
	h := &Http{
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
func (h *Http) SetHeader(key, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.headers[key] = value
}

func (h *Http) snapshotHeaders() map[string]string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[string]string, len(h.headers))
	for k, v := range h.headers {
		out[k] = v
	}
	return out
}

// Get sends a GET request to path and JSON-decodes the response into out.
func (h *Http) Get(ctx context.Context, path string, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodGet, path, nil, out)
}

// Post sends a POST request with a JSON-encoded body and decodes the response into out.
func (h *Http) Post(ctx context.Context, path string, body, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodPost, path, body, out)
}

// Put sends a PUT request with a JSON-encoded body and decodes the response into out.
func (h *Http) Put(ctx context.Context, path string, body, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodPut, path, body, out)
}

// Patch sends a PATCH request with a JSON-encoded body and decodes the response into out.
func (h *Http) Patch(ctx context.Context, path string, body, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodPatch, path, body, out)
}

// Delete sends a DELETE request to path and JSON-decodes the response into out.
func (h *Http) Delete(ctx context.Context, path string, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodDelete, path, nil, out)
}

// FireAndForget sends a request in the background, logging any error if a
// logger was configured via WithLogger. The caller's context values are kept
// but its cancellation/deadline are dropped so the request outlives the caller.
func (h *Http) FireAndForget(ctx context.Context, method, path string, body interface{}) {
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		if _, err := h.request(bgCtx, method, path, body, nil); err != nil && h.logger != nil {
			h.logger.Error(err, "fire and forget request failed", "method", method, "path", path)
		}
	}()
}

func (h *Http) request(ctx context.Context, method, path string, body, out interface{}) (int, error) {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		payload = b
	}

	headers := h.snapshotHeaders()
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

		code, respBody, err := h.doOnce(ctx, method, path, payload, headers)
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
			httpErr := &HTTPError{Code: code, Body: respBody}
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

	if httpErr, ok := lastErr.(*HTTPError); ok {
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

func (h *Http) doOnce(ctx context.Context, method, path string, payload []byte, headers map[string]string) (int, []byte, error) {
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
		req.Header.SetContentType("application/json")
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
