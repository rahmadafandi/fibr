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
		done <- run(app, "127.0.0.1:0", 5*time.Second, shutdown, c1, c2)
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
		done <- run(app, "127.0.0.1:0", 5*time.Second, shutdown,
			func(ctx context.Context) error { return assert.AnError })
	}()
	time.Sleep(100 * time.Millisecond)
	close(shutdown)

	err := <-done
	assert.Error(t, err)
}

func TestRunListenError(t *testing.T) {
	app := fiber.New()
	shutdown := make(chan struct{})
	err := run(app, "256.256.256.256:99999", time.Second, shutdown)
	assert.Error(t, err)
}
