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
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestHttp(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		err := fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Method()) {
			case fasthttp.MethodGet:
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"message": "get success"}`))
			case fasthttp.MethodPost:
				var reqBody map[string]interface{}
				json.Unmarshal(ctx.PostBody(), &reqBody)
				assert.Equal(t, "test", reqBody["data"])
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"message": "post success"}`))
			case fasthttp.MethodPut:
				var reqBody map[string]interface{}
				json.Unmarshal(ctx.PostBody(), &reqBody)
				assert.Equal(t, "test", reqBody["data"])
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"message": "put success"}`))
			case fasthttp.MethodDelete:
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetBody([]byte(`{"message": "delete success"}`))
			}
		})
		assert.NoError(t, err)
	}()

	h := New("http://localhost")
	h.Client = &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	h.SetHeader("Authorization", "Bearer token")

	t.Run("GET", func(t *testing.T) {
		var resp map[string]interface{}
		err := h.Get("/", &resp)
		assert.NoError(t, err)
		assert.Equal(t, "get success", resp["message"])
	})

	t.Run("POST", func(t *testing.T) {
		var resp map[string]interface{}
		err := h.Post("/", map[string]interface{}{"data": "test"}, &resp)
		assert.NoError(t, err)
		assert.Equal(t, "post success", resp["message"])
	})

	t.Run("PUT", func(t *testing.T) {
		var resp map[string]interface{}
		err := h.Put("/", map[string]interface{}{"data": "test"}, &resp)
		assert.NoError(t, err)
		assert.Equal(t, "put success", resp["message"])
	})

	t.Run("PATCH", func(t *testing.T) {
		var resp map[string]interface{}
		err := h.Patch("/", map[string]interface{}{"data": "test"}, &resp)
		assert.NoError(t, err)
		assert.Equal(t, "patch success", resp["message"])
	})

	t.Run("DELETE", func(t *testing.T) {
		var resp map[string]interface{}
		err := h.Delete("/", &resp)
		assert.NoError(t, err)
		assert.Equal(t, "delete success", resp["message"])
	})

	t.Run("FireAndForget", func(t *testing.T) {
		done := make(chan bool)

		ln := fasthttputil.NewInmemoryListener()
		defer ln.Close()

		go func() {
			err := fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
				done <- true
			})
			assert.NoError(t, err)
		}()

		h := New("http://localhost")
		h.Client = &fasthttp.Client{
			Dial: func(addr string) (net.Conn, error) {
				return ln.Dial()
			},
		}

		h.FireAndForget(fasthttp.MethodPost, "/", map[string]interface{}{"data": "test"})

		<-done
	})
}