// Copyright 2026 Rahmad Afandi. MIT License.

package http

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func dialClient(ln *fasthttputil.InmemoryListener) *fasthttp.Client {
	return &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) { return ln.Dial() },
	}
}

func TestHttp(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		_ = fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Method()) {
			case fasthttp.MethodGet:
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"message":"get success"}`))
			case fasthttp.MethodPost:
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"message":"post success"}`))
			case fasthttp.MethodPut:
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"message":"put success"}`))
			case fasthttp.MethodPatch:
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"message":"patch success"}`))
			case fasthttp.MethodDelete:
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"message":"delete success"}`))
			}
		})
	}()

	h := New("http://localhost", WithClient(dialClient(ln)), WithHeader("Authorization", "Bearer token"))
	ctx := context.Background()

	t.Run("GET", func(t *testing.T) {
		var resp map[string]interface{}
		code, err := h.Get(ctx, "/", &resp)
		assert.NoError(t, err)
		assert.Equal(t, 200, code)
		assert.Equal(t, "get success", resp["message"])
	})

	t.Run("POST", func(t *testing.T) {
		var resp map[string]interface{}
		code, err := h.Post(ctx, "/", map[string]interface{}{"data": "test"}, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 200, code)
		assert.Equal(t, "post success", resp["message"])
	})

	t.Run("DELETE", func(t *testing.T) {
		var resp map[string]interface{}
		code, err := h.Delete(ctx, "/", &resp)
		assert.NoError(t, err)
		assert.Equal(t, 200, code)
		assert.Equal(t, "delete success", resp["message"])
	})

	t.Run("PUT", func(t *testing.T) {
		var resp map[string]interface{}
		code, err := h.Put(ctx, "/", map[string]interface{}{"data": "test"}, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 200, code)
		assert.Equal(t, "put success", resp["message"])
	})

	t.Run("PATCH", func(t *testing.T) {
		var resp map[string]interface{}
		code, err := h.Patch(ctx, "/", map[string]interface{}{"data": "test"}, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 200, code)
		assert.Equal(t, "patch success", resp["message"])
	})
}

func TestHttpNon2xx(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		_ = fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			ctx.SetBody([]byte(`not found`))
		})
	}()

	h := New("http://localhost", WithClient(dialClient(ln)))

	var resp map[string]interface{}
	code, err := h.Get(context.Background(), "/missing", &resp)
	assert.Equal(t, 404, code)
	assert.Error(t, err)

	var httpErr *Error
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, 404, httpErr.Code)
	assert.Equal(t, "not found", string(httpErr.Body))
}

func TestHttpNoRetryOn4xx(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	var hits int32
	go func() {
		_ = fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
			atomic.AddInt32(&hits, 1)
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			ctx.SetBody([]byte(`bad`))
		})
	}()

	h := New("http://localhost", WithClient(dialClient(ln)), WithRetry(3, time.Millisecond))
	code, err := h.Get(context.Background(), "/", nil)
	assert.Equal(t, 400, code)
	assert.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits)) // 4xx must not be retried
}

func TestFireAndForget(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	got := make(chan string, 1)
	go func() {
		_ = fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
			got <- string(ctx.Method())
			ctx.SetStatusCode(fasthttp.StatusOK)
		})
	}()

	h := New("http://localhost", WithClient(dialClient(ln)))

	// Cancel the caller context immediately; FireAndForget must still send.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.FireAndForget(ctx, Post, "/", map[string]interface{}{"data": "test"})

	select {
	case m := <-got:
		assert.Equal(t, "POST", m)
	case <-time.After(2 * time.Second):
		t.Fatal("FireAndForget request never reached the server")
	}
}

func TestHttpRetry(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	var hits int32
	go func() {
		_ = fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
			n := atomic.AddInt32(&hits, 1)
			if n < 3 {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				return
			}
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetBody([]byte(`{"ok":true}`))
		})
	}()

	// 1 initial + 2 retries = 3 attempts; the 3rd succeeds.
	h := New("http://localhost", WithClient(dialClient(ln)), WithRetry(2, time.Millisecond))

	var resp map[string]interface{}
	code, err := h.Get(context.Background(), "/", &resp)
	assert.NoError(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, int32(3), atomic.LoadInt32(&hits))
}
