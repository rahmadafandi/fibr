// Copyright 2026 Rahmad Afandi. MIT License.

package http

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestBreakerFailureClassification(t *testing.T) {
	assert.True(t, breakerFailure(0, context.DeadlineExceeded)) // transport error
	assert.True(t, breakerFailure(500, &Error{Code: 500}))      // server error
	assert.True(t, breakerFailure(503, nil))
	assert.False(t, breakerFailure(404, &Error{Code: 404})) // client error
	assert.False(t, breakerFailure(200, nil))               // success
	assert.False(t, breakerFailure(200, context.Canceled))  // decode error on 2xx
}

func TestBreakerOpensAfterFailuresAndRecovers(t *testing.T) {
	b := newBreaker(2, 50*time.Millisecond)
	now := time.Unix(0, 0)
	b.now = func() time.Time { return now }

	assert.True(t, b.allow())
	b.onFailure()
	assert.True(t, b.allow()) // 1 failure, still closed
	b.onFailure()             // 2nd consecutive failure -> open
	assert.False(t, b.allow())

	// Before the timeout, still open.
	now = now.Add(40 * time.Millisecond)
	assert.False(t, b.allow())

	// After the timeout, a single probe is allowed (half-open).
	now = now.Add(20 * time.Millisecond)
	assert.True(t, b.allow())
	assert.False(t, b.allow()) // only one probe

	// Probe succeeds -> closed.
	b.onSuccess()
	assert.True(t, b.allow())
}

func TestBreakerHalfOpenFailureReopens(t *testing.T) {
	b := newBreaker(1, 10*time.Millisecond)
	now := time.Unix(0, 0)
	b.now = func() time.Time { return now }

	b.onFailure() // -> open
	now = now.Add(20 * time.Millisecond)
	assert.True(t, b.allow()) // half-open probe
	b.onFailure()             // probe fails -> open again
	assert.False(t, b.allow())
}

func TestHTTPCircuitBreakerRejectsWhenOpen(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	var hits int32
	go func() {
		_ = fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
			atomic.AddInt32(&hits, 1)
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		})
	}()

	h := New("http://localhost",
		WithClient(dialClient(ln)),
		WithCircuitBreaker(2, time.Minute),
	)
	ctx := context.Background()

	// Two 5xx responses trip the breaker.
	_, err := h.Get(ctx, "/", nil)
	assert.Error(t, err)
	_, err = h.Get(ctx, "/", nil)
	assert.Error(t, err)

	hitsAfterTrip := atomic.LoadInt32(&hits)

	// Next call is rejected without reaching the server.
	_, err = h.Get(ctx, "/", nil)
	assert.ErrorIs(t, err, ErrCircuitOpen)
	assert.Equal(t, hitsAfterTrip, atomic.LoadInt32(&hits), "request must not be sent when open")
}
