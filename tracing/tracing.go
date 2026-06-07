// Copyright 2026 Rahmad Afandi. MIT License.

// Package tracing wires OpenTelemetry distributed tracing for Fiber apps: it
// builds an OTLP/HTTP tracer provider (configured by the standard OTEL_ env
// vars), installs it and the W3C propagator as the OTel globals, and returns a
// shutdown function for graceful shutdown.
package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	serviceName string
	sampler     sdktrace.Sampler
}

// Option configures tracer setup.
type Option func(*config)

// WithServiceName sets the resource service.name used when OTEL_SERVICE_NAME is
// not set in the environment.
func WithServiceName(name string) Option {
	return func(c *config) { c.serviceName = name }
}

// WithSampler overrides the default ParentBased(AlwaysSample) sampler.
func WithSampler(s sdktrace.Sampler) Option {
	return func(c *config) { c.sampler = s }
}

// Setup builds an OTLP/HTTP tracer provider, installs it and the W3C trace
// context + baggage propagator as the OTel globals, and returns a shutdown
// function that flushes and stops the provider. Exporter endpoint and headers
// come from the standard OTEL_ env vars (e.g. OTEL_EXPORTER_OTLP_ENDPOINT).
func Setup(ctx context.Context, opts ...Option) (func(context.Context) error, error) {
	cfg := config{
		serviceName: "fiber-app",
		sampler:     sdktrace.ParentBased(sdktrace.AlwaysSample()),
	}
	for _, o := range opts {
		o(&cfg)
	}

	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}

	// Explicit service.name first, env (OTEL_SERVICE_NAME) overrides it.
	res, err := resource.New(ctx,
		resource.WithAttributes(attribute.String("service.name", cfg.serviceName)),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(cfg.sampler),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp.Shutdown, nil
}

// LogFields returns the trace and span IDs from ctx's span context. ok is false
// when ctx carries no valid span context.
func LogFields(ctx context.Context) (traceID, spanID string, ok bool) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return "", "", false
	}
	return sc.TraceID().String(), sc.SpanID().String(), true
}
