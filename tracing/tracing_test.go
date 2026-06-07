// Copyright 2026 Rahmad Afandi. MIT License.

package tracing_test

import (
	"context"
	"testing"

	"github.com/rahmadafandi/fiber-helpers/tracing"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestSetupInstallsProviderAndPropagator(t *testing.T) {
	shutdown, err := tracing.Setup(context.Background(), tracing.WithServiceName("test-svc"))
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	_, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, ok, "global provider should be the SDK provider")

	require.Contains(t, otel.GetTextMapPropagator().Fields(), "traceparent")
}

func TestLogFields(t *testing.T) {
	_, _, ok := tracing.LogFields(context.Background())
	require.False(t, ok)

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("t").Start(context.Background(), "op")
	defer span.End()

	tid, sid, ok := tracing.LogFields(ctx)
	require.True(t, ok)
	require.Len(t, tid, 32)
	require.Len(t, sid, 16)
}
