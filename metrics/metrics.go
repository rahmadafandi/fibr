// Copyright 2026 Rahmad Afandi. MIT License.

// Package metrics provides Prometheus request metrics for Fiber apps: a
// middleware that records request count and latency, and a handler that serves
// the Prometheus exposition format. It is a thin wrapper over
// prometheus/client_golang's default registry (which also exposes the standard
// Go-runtime and process collectors).
package metrics

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsPath is the conventional scrape path. Middleware skips it so scrapes
// are not self-counted.
const MetricsPath = "/metrics"

type config struct {
	namespace string
}

// Option configures the metrics middleware.
type Option func(*config)

// WithNamespace prefixes all metric names with namespace + "_". Because the
// metric vectors are registered once (on the first Middleware call), the
// namespace is captured from that first call and fixed thereafter.
func WithNamespace(ns string) Option {
	return func(c *config) { c.namespace = ns }
}

var (
	once     sync.Once
	reqTotal *prometheus.CounterVec
	reqDur   *prometheus.HistogramVec
)

// register builds and registers the metric vectors exactly once.
func register(cfg config) {
	once.Do(func() {
		labels := []string{"method", "path", "status"}
		reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: cfg.namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests processed.",
		}, labels)
		reqDur = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, labels)
	})
}

// Middleware returns a Fiber handler that records request count and duration,
// labeled by method, route template (e.g. "/items/:id"), and status.
func Middleware(opts ...Option) fiber.Handler {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}
	register(cfg)
	return func(c *fiber.Ctx) error {
		if c.Path() == MetricsPath {
			return c.Next()
		}
		start := time.Now()
		err := c.Next()
		path := c.Path()
		if r := c.Route(); r != nil && r.Path != "" {
			path = r.Path
		}
		// Resolve the status the client will actually receive. When a handler
		// returns an error, Fiber's error handler sets the response code only
		// after the middleware chain unwinds, so c.Response().StatusCode() is
		// still the default here — derive it from the error instead.
		code := c.Response().StatusCode()
		if err != nil {
			code = fiber.StatusInternalServerError
			var fe *fiber.Error
			if errors.As(err, &fe) {
				code = fe.Code
			}
		}
		status := strconv.Itoa(code)
		method := c.Method()
		reqTotal.WithLabelValues(method, path, status).Inc()
		reqDur.WithLabelValues(method, path, status).Observe(time.Since(start).Seconds())
		return err
	}
}

// Handler serves the Prometheus exposition format (default registry) as a Fiber
// handler.
func Handler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}
