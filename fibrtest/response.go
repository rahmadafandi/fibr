// Copyright 2026 Rahmad Afandi. MIT License.

package fibrtest

import (
	"encoding/json"
	"net/http"
)

// Response is an executed request's result. Body is read once so it can be
// inspected repeatedly.
type Response struct {
	t    TB
	Raw  *http.Response
	Body []byte
}

// Status returns the HTTP status code.
func (r *Response) Status() int {
	if r.Raw == nil {
		return 0
	}
	return r.Raw.StatusCode
}

// ExpectStatus calls Fatalf unless the status equals code. It returns the
// Response for chaining.
func (r *Response) ExpectStatus(code int) *Response {
	r.t.Helper()
	if got := r.Status(); got != code {
		r.t.Fatalf("fibrtest: expected status %d, got %d (body: %s)", code, got, r.Body)
	}
	return r
}

// JSON decodes the body into out, calling Fatalf on error. It returns the
// Response for chaining.
func (r *Response) JSON(out any) *Response {
	r.t.Helper()
	if err := json.Unmarshal(r.Body, out); err != nil {
		r.t.Fatalf("fibrtest: decode JSON body: %v (body: %s)", err, r.Body)
	}
	return r
}

// Text returns the body as a string.
func (r *Response) Text() string {
	return string(r.Body)
}
