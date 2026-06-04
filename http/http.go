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

func WithTimeout(d time.Duration) Option { return func(h *Http) { h.timeout = d } }
func WithRetry(n int, backoff time.Duration) Option {
	return func(h *Http) { h.retries = n; h.backoff = backoff }
}
func WithHeader(key, value string) Option  { return func(h *Http) { h.headers[key] = value } }
func WithClient(c *fasthttp.Client) Option { return func(h *Http) { h.Client = c } }
func WithLogger(l *logger.Logger) Option   { return func(h *Http) { h.logger = l } }

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

func (h *Http) Get(ctx context.Context, path string, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodGet, path, nil, out)
}

func (h *Http) Post(ctx context.Context, path string, body, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodPost, path, body, out)
}

func (h *Http) Put(ctx context.Context, path string, body, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodPut, path, body, out)
}

func (h *Http) Patch(ctx context.Context, path string, body, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodPatch, path, body, out)
}

func (h *Http) Delete(ctx context.Context, path string, out interface{}) (int, error) {
	return h.request(ctx, fasthttp.MethodDelete, path, nil, out)
}

// FireAndForget sends a request in the background, logging any error if a
// logger was configured via WithLogger.
func (h *Http) FireAndForget(ctx context.Context, method, path string, body interface{}) {
	go func() {
		if _, err := h.request(ctx, method, path, body, nil); err != nil && h.logger != nil {
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
	var lastErr error

	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}

		code, respBody, err := h.doOnce(ctx, method, path, payload, headers)
		if err != nil {
			lastErr = err
			if i < attempts-1 && h.backoff > 0 {
				time.Sleep(h.backoff)
			}
			continue
		}

		if code < 200 || code >= 300 {
			lastErr = &HTTPError{Code: code, Body: respBody}
			if i < attempts-1 && h.backoff > 0 {
				time.Sleep(h.backoff)
			}
			continue
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
	return 0, lastErr
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
