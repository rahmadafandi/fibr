// Copyright 2026 Rahmad Afandi. MIT License.

// Package dbresolver provides an explicit read/write split over Bun: writes go
// to the primary, reads are spread round-robin across replicas. Routing is
// explicit (Reader/Writer) because Bun query hooks are observational and cannot
// redirect a query's connection.
package dbresolver

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/uptrace/bun"
)

// Resolver routes database access to a primary and read replicas.
type Resolver struct {
	primary  *bun.DB
	replicas []*bun.DB
	counter  atomic.Uint64
}

// New returns a Resolver. Writes use primary; reads round-robin across replicas
// (falling back to primary when none are given).
func New(primary *bun.DB, replicas ...*bun.DB) *Resolver {
	return &Resolver{primary: primary, replicas: replicas}
}

// Writer returns the primary database, for writes and read-after-write reads.
func (r *Resolver) Writer() *bun.DB { return r.primary }

// Reader returns the next replica (round-robin), or the primary when no replicas
// are configured.
func (r *Resolver) Reader() *bun.DB {
	if len(r.replicas) == 0 {
		return r.primary
	}
	i := r.counter.Add(1) - 1
	return r.replicas[i%uint64(len(r.replicas))]
}

// Ping pings the primary and every replica, returning the joined errors.
func (r *Resolver) Ping(ctx context.Context) error {
	var errs []error
	if err := r.primary.PingContext(ctx); err != nil {
		errs = append(errs, err)
	}
	for _, rep := range r.replicas {
		if err := rep.PingContext(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Close closes the primary and every replica, returning the joined errors.
func (r *Resolver) Close() error {
	var errs []error
	if err := r.primary.Close(); err != nil {
		errs = append(errs, err)
	}
	for _, rep := range r.replicas {
		if err := rep.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
