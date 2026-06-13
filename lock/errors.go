// Copyright 2026 Rahmad Afandi. MIT License.

// Package lock provides a single-instance Redis distributed mutex with
// owner-only release and TTL extension, for single-execution across replicas.
package lock

import "errors"

// ErrNotAcquired is returned when a lock could not be obtained: by Acquire when
// the context ends before the lock is free, and by Do when another owner holds
// the lock (the wrapped function is not run).
var ErrNotAcquired = errors.New("lock: not acquired")

// ErrNotHeld is returned by Release/Extend when the lock is no longer owned by
// this handle (it expired or was acquired by another owner).
var ErrNotHeld = errors.New("lock: not held")
