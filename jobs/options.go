// Copyright 2026 Rahmad Afandi. MIT License.

package jobs

import (
	"time"

	"github.com/hibiken/asynq"
)

// WithRetry sets the maximum number of retries for an enqueued task.
func WithRetry(n int) asynq.Option { return asynq.MaxRetry(n) }

// WithPriority routes the task to the named queue. asynq processes queues by
// weight, so distinct queues act as priority levels.
func WithPriority(queue string) asynq.Option { return asynq.Queue(queue) }

// WithProcessIn delays processing of the task by d.
func WithProcessIn(d time.Duration) asynq.Option { return asynq.ProcessIn(d) }

// WithDeadline sets a hard deadline after which the task is no longer processed.
func WithDeadline(t time.Time) asynq.Option { return asynq.Deadline(t) }
