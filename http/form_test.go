// Copyright 2026 Rahmad Afandi. MIT License.

package http

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestPostFormAndMultipart(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		_ = fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
			ct := string(ctx.Request.Header.ContentType())
			switch {
			case strings.HasPrefix(ct, "application/x-www-form-urlencoded"):
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"name":"` + string(ctx.PostArgs().Peek("name")) + `"}`))
			case strings.HasPrefix(ct, "multipart/form-data"):
				field := string(ctx.FormValue("field"))
				var content string
				if fh, err := ctx.FormFile("file"); err == nil {
					f, _ := fh.Open()
					b, _ := io.ReadAll(f)
					_ = f.Close()
					content = string(b)
				}
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"field":"` + field + `","file":"` + content + `"}`))
			default:
				ctx.SetStatusCode(fasthttp.StatusUnsupportedMediaType)
			}
		})
	}()

	h := New("http://localhost", WithClient(dialClient(ln)))
	ctx := context.Background()

	t.Run("PostForm", func(t *testing.T) {
		var out map[string]string
		code, err := h.PostForm(ctx, "/f", url.Values{"name": {"sam"}}, &out)
		require.NoError(t, err)
		assert.Equal(t, 200, code)
		assert.Equal(t, "sam", out["name"])
	})

	t.Run("PostMultipart", func(t *testing.T) {
		var out map[string]string
		code, err := h.PostMultipart(ctx, "/m",
			map[string]string{"field": "v"},
			[]FileField{{Field: "file", Filename: "a.txt", Content: strings.NewReader("hello")}},
			&out)
		require.NoError(t, err)
		assert.Equal(t, 200, code)
		assert.Equal(t, "v", out["field"])
		assert.Equal(t, "hello", out["file"])
	})
}
