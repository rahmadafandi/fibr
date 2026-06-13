// Copyright 2026 Rahmad Afandi. MIT License.

package fibrtest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

// Request builds a single HTTP request against the Client's app.
type Request struct {
	c           *Client
	method      string
	path        string
	header      http.Header
	query       url.Values
	body        []byte
	contentType string
}

// JSON sets the body to the JSON encoding of v with content type
// application/json. A marshal error calls Fatalf.
func (r *Request) JSON(v any) *Request {
	r.c.t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		r.c.t.Fatalf("fibrtest: marshal request body: %v", err)
		return r
	}
	r.body = b
	r.contentType = "application/json"
	return r
}

// Body sets a raw body and content type.
func (r *Request) Body(raw []byte, contentType string) *Request {
	r.body = raw
	r.contentType = contentType
	return r
}

// Header adds a per-request header (overrides the Client default for that key).
func (r *Request) Header(key, value string) *Request {
	r.header.Set(key, value)
	return r
}

// Bearer sets the Authorization header for this request.
func (r *Request) Bearer(token string) *Request {
	r.header.Set("Authorization", "Bearer "+token)
	return r
}

// Query adds a query parameter.
func (r *Request) Query(key, value string) *Request {
	if r.query == nil {
		r.query = url.Values{}
	}
	r.query.Add(key, value)
	return r
}

// Do builds the request, runs it via app.Test, and returns the Response. A
// transport error calls Fatalf.
func (r *Request) Do() *Response {
	r.c.t.Helper()

	path := r.path
	if len(r.query) > 0 {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		path += sep + r.query.Encode()
	}

	var body io.Reader
	if r.body != nil {
		body = bytes.NewReader(r.body)
	}
	req := httptest.NewRequest(r.method, path, body)

	// Client defaults first, then per-request headers override.
	for k, vs := range r.c.headers {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	for k, vs := range r.header {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	if r.contentType != "" {
		req.Header.Set("Content-Type", r.contentType)
	}

	resp, err := r.c.app.Test(req, int(r.c.timeout.Milliseconds()))
	if err != nil {
		r.c.t.Fatalf("fibrtest: app.Test %s %s: %v", r.method, path, err)
		return &Response{t: r.c.t}
	}

	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		r.c.t.Fatalf("fibrtest: read response body: %v", err)
		return &Response{t: r.c.t, Raw: resp}
	}
	return &Response{t: r.c.t, Raw: resp, Body: raw}
}
