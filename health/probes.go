// Copyright 2026 Rahmad Afandi. MIT License.

package health

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/redis/go-redis/v9"
)

// PingRedis is a readiness check that pings a Redis server.
func PingRedis(client redis.UniversalClient) NamedCheck {
	return Check("redis", func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	})
}

// PingHTTP is a readiness check that issues a GET to url. A transport error or a
// 5xx status fails the check; a reachable dependency returning below 500 is
// considered up.
func PingHTTP(name, url string) NamedCheck {
	return Check(name, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode >= 500 {
			return fmt.Errorf("health: %s returned status %d", name, resp.StatusCode)
		}
		return nil
	})
}

// PingTCP is a readiness check that opens a TCP connection to addr.
func PingTCP(name, addr string) NamedCheck {
	return Check(name, func(ctx context.Context) error {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return err
		}
		return conn.Close()
	})
}
