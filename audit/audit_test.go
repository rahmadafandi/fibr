// Copyright 2026 Rahmad Afandi. MIT License.

package audit_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/audit"
)

// captureSink records the last entry it received.
type captureSink struct{ last *audit.Entry }

func (s *captureSink) Record(_ context.Context, e *audit.Entry) error {
	s.last = e
	return nil
}

func TestRecordUsesSinkAndSetsCreatedAt(t *testing.T) {
	sink := &captureSink{}
	rec := audit.New(sink)

	require.NoError(t, rec.Record(context.Background(), audit.Entry{Action: "x"}))
	require.NotNil(t, sink.last)
	assert.Equal(t, "x", sink.last.Action)
	assert.False(t, sink.last.CreatedAt.IsZero())
}

func TestFromRequestPrefillsContext(t *testing.T) {
	sink := &captureSink{}
	rec := audit.New(sink, audit.WithActor(func(c *fiber.Ctx) string {
		return c.Get("X-User")
	}))

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestid", "req-123") // what context.GetRequestID reads
		return c.Next()
	})
	app.Get("/", func(c *fiber.Ctx) error {
		e := rec.FromRequest(c)
		e.Action = "page.view"
		return rec.Record(c.UserContext(), e)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User", "alice")
	_, err := app.Test(req)
	require.NoError(t, err)

	require.NotNil(t, sink.last)
	assert.Equal(t, "alice", sink.last.Actor)
	assert.Equal(t, "req-123", sink.last.RequestID)
	assert.Equal(t, "page.view", sink.last.Action)
	assert.NotEmpty(t, sink.last.IP)
}
