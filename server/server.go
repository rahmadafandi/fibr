// Copyright 2026 Rahmad Afandi. MIT License.

// Package server runs a Fiber app with signal-based graceful shutdown.
package server

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Config tunes graceful shutdown. Timeout bounds the whole shutdown sequence.
// PreShutdown hooks run before the server stops accepting connections (e.g. to
// flip a readiness gate); DrainDelay then waits so a load balancer can stop
// routing before in-flight connections are closed; Cleanup hooks run after the
// server has shut down.
type Config struct {
	Timeout     time.Duration
	DrainDelay  time.Duration
	PreShutdown []func(ctx context.Context) error
	Cleanup     []func(ctx context.Context) error
}

// RunGraceful starts app and blocks until SIGINT/SIGTERM (or a listen error),
// then shuts the server down within timeout and runs the cleanup hooks in order.
func RunGraceful(app *fiber.App, addr string, timeout time.Duration,
	cleanup ...func(ctx context.Context) error) error {

	return RunGracefulWithConfig(app, addr, Config{Timeout: timeout, Cleanup: cleanup})
}

// RunGracefulWithConfig is RunGraceful with pre-shutdown hooks and a drain
// delay. See Config for the shutdown ordering.
func RunGracefulWithConfig(app *fiber.App, addr string, cfg Config) error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	shutdown := make(chan struct{})
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-stop:
			close(shutdown)
		case <-done:
		}
	}()

	return run(app, addr, shutdown, cfg)
}

// run is the testable core: it serves until shutdown is closed (or Listen
// errors), then runs pre-shutdown hooks, the drain delay, shutdown, and cleanup
// hooks in order.
func run(app *fiber.App, addr string, shutdown <-chan struct{}, cfg Config) error {
	listenErr := make(chan error, 1)
	go func() {
		if err := app.Listen(addr); err != nil {
			listenErr <- err
		}
	}()

	select {
	case err := <-listenErr:
		return err
	case <-shutdown:
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	var errs []error
	for _, fn := range cfg.PreShutdown {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if cfg.DrainDelay > 0 {
		t := time.NewTimer(cfg.DrainDelay)
		select {
		case <-t.C:
		case <-ctx.Done():
			t.Stop()
		}
	}
	if err := app.ShutdownWithContext(ctx); err != nil {
		errs = append(errs, err)
	}
	for _, fn := range cfg.Cleanup {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
