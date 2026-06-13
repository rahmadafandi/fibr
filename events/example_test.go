// Copyright 2026 Rahmad Afandi. MIT License.

package events_test

import (
	"context"
	"fmt"

	"github.com/rahmadafandi/fibr/events"
)

// Subscribe handlers for a typed event, then publish one. In the default sync
// mode handlers run inline and Publish returns their joined errors.
func ExamplePublish() {
	type OrderCreated struct{ ID int }

	bus := events.New()
	events.Subscribe(bus, func(_ context.Context, e OrderCreated) error {
		fmt.Println("email: order", e.ID)
		return nil
	})
	events.Subscribe(bus, func(_ context.Context, e OrderCreated) error {
		fmt.Println("analytics: order", e.ID)
		return nil
	})

	_ = events.Publish(context.Background(), bus, OrderCreated{ID: 7})
	// Output:
	// email: order 7
	// analytics: order 7
}
