// Copyright 2026 Rahmad Afandi. MIT License.

package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

type ridKey struct{}

func TestWithContextHeaderPropagates(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	gotHeader := make(chan string, 2)
	go func() {
		_ = fasthttp.Serve(ln, func(c *fasthttp.RequestCtx) {
			gotHeader <- string(c.Request.Header.Peek("X-Request-ID"))
			c.SetStatusCode(fasthttp.StatusOK)
		})
	}()

	h := New("http://localhost",
		WithClient(dialClient(ln)),
		WithContextHeader("X-Request-ID", func(ctx context.Context) string {
			v, _ := ctx.Value(ridKey{}).(string)
			return v
		}),
	)

	// Context carries a request id -> forwarded.
	ctx := context.WithValue(context.Background(), ridKey{}, "rid-123")
	_, err := h.Get(ctx, "/", nil)
	require.NoError(t, err)
	assert.Equal(t, "rid-123", <-gotHeader)

	// No request id in context -> header absent.
	_, err = h.Get(context.Background(), "/", nil)
	require.NoError(t, err)
	assert.Equal(t, "", <-gotHeader)
}
