// Copyright 2026 Rahmad Afandi. MIT License.

// Package events provides a generic, type-keyed in-process event bus so domain
// event producers and handlers are decoupled within a process. It complements
// the durable, cross-process outbox package.
package events

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/rahmadafandi/fibr/logger"
)

// Bus is an in-process event bus. Handlers are keyed by event type.
type Bus struct {
	mu       sync.RWMutex
	handlers map[reflect.Type][]any
	async    bool
	logger   *logger.Logger
}

// Option configures a Bus.
type Option func(*Bus)

// WithAsync dispatches each handler in its own goroutine, so Publish returns
// immediately and handler errors are logged rather than returned.
func WithAsync() Option {
	return func(b *Bus) { b.async = true }
}

// WithLogger sets the logger used for async handler errors.
func WithLogger(l *logger.Logger) Option {
	return func(b *Bus) { b.logger = l }
}

// New returns an event bus. By default it delivers synchronously.
func New(opts ...Option) *Bus {
	b := &Bus{handlers: make(map[reflect.Type][]any)}
	for _, opt := range opts {
		opt(b)
	}
	if b.logger == nil {
		b.logger = logger.Default()
	}
	return b
}

// Subscribe registers handler for events of type T.
func Subscribe[T any](b *Bus, handler func(context.Context, T) error) {
	key := reflect.TypeFor[T]()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[key] = append(b.handlers[key], handler)
}

// Publish delivers evt to all handlers registered for type T. In sync mode
// (default) it runs them in registration order and returns their joined errors;
// in async mode it dispatches each in a goroutine and returns nil.
func Publish[T any](ctx context.Context, b *Bus, evt T) error {
	key := reflect.TypeFor[T]()

	b.mu.RLock()
	stored := b.handlers[key]
	handlers := make([]func(context.Context, T) error, len(stored))
	for i, h := range stored {
		handlers[i] = h.(func(context.Context, T) error)
	}
	b.mu.RUnlock()

	if b.async {
		for _, fn := range handlers {
			go func(fn func(context.Context, T) error) {
				if err := fn(ctx, evt); err != nil {
					b.logger.Error(err, "events: async handler failed", "type", key.String())
				}
			}(fn)
		}
		return nil
	}

	var errs []error
	for _, fn := range handlers {
		if err := fn(ctx, evt); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
