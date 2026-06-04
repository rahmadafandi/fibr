// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
