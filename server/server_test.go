// Copyright 2026 Rahmad Afandi. MIT License.

package server

import (
	"context"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRunCleanupOrder(t *testing.T) {
	app := fiber.New()
	shutdown := make(chan struct{})

	var order []string
	c1 := func(ctx context.Context) error { order = append(order, "c1"); return nil }
	c2 := func(ctx context.Context) error { order = append(order, "c2"); return nil }

	done := make(chan error, 1)
	go func() {
		done <- run(app, "127.0.0.1:0", shutdown, Config{Timeout: 5 * time.Second, Cleanup: []func(context.Context) error{c1, c2}})
	}()

	time.Sleep(100 * time.Millisecond)
	close(shutdown)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after shutdown")
	}
	assert.Equal(t, []string{"c1", "c2"}, order)
}

func TestRunCleanupErrorReturned(t *testing.T) {
	app := fiber.New()
	shutdown := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		done <- run(app, "127.0.0.1:0", shutdown, Config{
			Timeout: 5 * time.Second,
			Cleanup: []func(context.Context) error{func(ctx context.Context) error { return assert.AnError }},
		})
	}()
	time.Sleep(100 * time.Millisecond)
	close(shutdown)

	err := <-done
	assert.Error(t, err)
}

func TestRunListenError(t *testing.T) {
	app := fiber.New()
	shutdown := make(chan struct{})
	err := run(app, "256.256.256.256:99999", shutdown, Config{Timeout: time.Second})
	assert.Error(t, err)
}

func TestRunPreShutdownRunsBeforeCleanup(t *testing.T) {
	app := fiber.New()
	shutdown := make(chan struct{})

	var order []string
	cfg := Config{
		Timeout:    5 * time.Second,
		DrainDelay: 10 * time.Millisecond,
		PreShutdown: []func(context.Context) error{
			func(context.Context) error { order = append(order, "pre"); return nil },
		},
		Cleanup: []func(context.Context) error{
			func(context.Context) error { order = append(order, "cleanup"); return nil },
		},
	}

	done := make(chan error, 1)
	go func() { done <- run(app, "127.0.0.1:0", shutdown, cfg) }()
	time.Sleep(100 * time.Millisecond)
	close(shutdown)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after shutdown")
	}
	assert.Equal(t, []string{"pre", "cleanup"}, order)
}
