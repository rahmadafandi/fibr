// Copyright 2026 Rahmad Afandi. MIT License.

package health_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rahmadafandi/fibr/health"
)

func TestReadinessGate(t *testing.T) {
	g := health.NewReadinessGate()
	check := g.Check()
	assert.Equal(t, "readiness", check.Name)

	// Open by default -> passes.
	assert.True(t, g.Ready())
	assert.NoError(t, check.Fn(context.Background()))

	// Closed -> fails.
	g.Close()
	assert.False(t, g.Ready())
	assert.Error(t, check.Fn(context.Background()))

	// Reopened -> passes again.
	g.Open()
	assert.NoError(t, check.Fn(context.Background()))
}
