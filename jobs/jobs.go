// Copyright 2026 Rahmad Afandi. MIT License.

// Package jobs provides a thin, typed wrapper over hibiken/asynq for enqueuing
// and processing background jobs, plus a mountable asynqmon monitoring handler.
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
)

// RedisConnOpt parses a redis:// URL into an asynq connection option.
func RedisConnOpt(redisURL string) (asynq.RedisConnOpt, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("jobs: invalid redis url: %w", err)
	}
	return opt, nil
}

// Client enqueues background jobs.
type Client struct {
	inner *asynq.Client
}

// NewClient creates a job client from a Redis connection option.
func NewClient(opt asynq.RedisConnOpt) *Client {
	return &Client{inner: asynq.NewClient(opt)}
}

// Enqueue JSON-marshals payload and enqueues a task of the given type.
func (c *Client) Enqueue(ctx context.Context, typename string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("jobs: encode %s payload: %w", typename, err)
	}
	return c.inner.EnqueueContext(ctx, asynq.NewTask(typename, b), opts...)
}

// Close releases the client's Redis connections.
func (c *Client) Close() error { return c.inner.Close() }

// ServerConfig configures the worker server.
type ServerConfig struct {
	Concurrency int
	Queues      map[string]int // optional; nil uses the asynq default
}

// Server processes background jobs.
type Server struct {
	inner *asynq.Server
	mux   *asynq.ServeMux
}

// NewServer creates a worker server from a Redis connection option.
func NewServer(opt asynq.RedisConnOpt, cfg ServerConfig) *Server {
	return &Server{
		inner: asynq.NewServer(opt, asynq.Config{Concurrency: cfg.Concurrency, Queues: cfg.Queues}),
		mux:   asynq.NewServeMux(),
	}
}

// Handle registers a typed handler for typename. It JSON-unmarshals the task
// payload into T before calling fn. A decode failure wraps asynq.SkipRetry so a
// malformed payload is not retried forever. It is a generic free function
// because Go methods cannot have type parameters.
func Handle[T any](s *Server, typename string, fn func(ctx context.Context, payload T) error) {
	s.mux.HandleFunc(typename, func(ctx context.Context, t *asynq.Task) error {
		var p T
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: decode %s payload: %v: %w", typename, err, asynq.SkipRetry)
		}
		return fn(ctx, p)
	})
}

// Run starts processing and blocks until the process receives SIGTERM/SIGINT.
func (s *Server) Run() error { return s.inner.Run(s.mux) }

// ProcessTask routes a task to its registered handler. Exposed so handlers can
// be unit tested without a running server.
func (s *Server) ProcessTask(ctx context.Context, t *asynq.Task) error {
	return s.mux.ProcessTask(ctx, t)
}

// MonitoringHandler returns the asynqmon UI handler rooted at rootPath.
func MonitoringHandler(opt asynq.RedisConnOpt, rootPath string) http.Handler {
	return asynqmon.New(asynqmon.Options{RootPath: rootPath, RedisConnOpt: opt})
}
