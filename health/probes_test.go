// Copyright 2026 Rahmad Afandi. MIT License.

package health_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/health"
)

func TestPingRedis(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	check := health.PingRedis(client)
	assert.Equal(t, "redis", check.Name)
	assert.NoError(t, check.Fn(context.Background()))

	mr.Close()
	assert.Error(t, check.Fn(context.Background()))
}

func TestPingHTTP(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ok.Close)
	assert.NoError(t, health.PingHTTP("api", ok.URL).Fn(context.Background()))

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(bad.Close)
	assert.Error(t, health.PingHTTP("api", bad.URL).Fn(context.Background()))

	assert.Error(t, health.PingHTTP("api", "http://127.0.0.1:0").Fn(context.Background()))
}

func TestPingTCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	check := health.PingTCP("svc", ln.Addr().String())
	assert.Equal(t, "svc", check.Name)
	assert.NoError(t, check.Fn(context.Background()))

	ln.Close()
	assert.Error(t, health.PingTCP("svc", ln.Addr().String()).Fn(context.Background()))
}
