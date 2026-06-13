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

// RunGraceful starts app and blocks until SIGINT/SIGTERM (or a listen error),
// then shuts the server down within timeout and runs the cleanup hooks in order.
func RunGraceful(app *fiber.App, addr string, timeout time.Duration,
	cleanup ...func(ctx context.Context) error) error {

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

	return run(app, addr, timeout, shutdown, cleanup...)
}

// run is the testable core: it serves until shutdown is closed (or Listen
// errors), then shuts down and runs cleanup hooks in order.
func run(app *fiber.App, addr string, timeout time.Duration, shutdown <-chan struct{},
	cleanup ...func(ctx context.Context) error) error {

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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var errs []error
	if err := app.ShutdownWithContext(ctx); err != nil {
		errs = append(errs, err)
	}
	for _, fn := range cleanup {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
