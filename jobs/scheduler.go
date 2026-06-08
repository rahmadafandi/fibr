// Copyright 2026 Rahmad Afandi. MIT License.

package jobs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Scheduler enqueues registered tasks on a cron schedule. Run exactly ONE
// instance: workers scale horizontally, schedulers do not — two schedulers
// enqueue every task twice.
type Scheduler struct {
	inner *asynq.Scheduler
}

// SchedulerOption configures a Scheduler.
type SchedulerOption func(*asynq.SchedulerOpts)

// WithLocation sets the time zone that cron specs are interpreted in. The
// default (nil or unset) is UTC.
func WithLocation(loc *time.Location) SchedulerOption {
	return func(o *asynq.SchedulerOpts) { o.Location = loc }
}

// NewScheduler creates a Scheduler from a Redis connection option.
func NewScheduler(opt asynq.RedisConnOpt, opts ...SchedulerOption) *Scheduler {
	o := &asynq.SchedulerOpts{}
	for _, fn := range opts {
		fn(o)
	}
	return &Scheduler{inner: asynq.NewScheduler(opt, o)}
}

// Register schedules a task: at each tick of cronspec, payload is JSON-marshaled
// and enqueued as a task of typename, processed by the worker's Handle[T]. It
// returns an entry id. Mirrors Client.Enqueue.
func (s *Scheduler) Register(cronspec, typename string, payload any, opts ...asynq.Option) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("jobs: encode %s payload: %w", typename, err)
	}
	return s.inner.Register(cronspec, asynq.NewTask(typename, b), opts...)
}

// Unregister removes a previously registered entry by the id returned from
// Register, so it stops being enqueued.
func (s *Scheduler) Unregister(entryID string) error { return s.inner.Unregister(entryID) }

// Run starts the scheduler and blocks until the process receives SIGTERM/SIGINT.
func (s *Scheduler) Run() error { return s.inner.Run() }

// Shutdown stops the scheduler gracefully.
func (s *Scheduler) Shutdown() { s.inner.Shutdown() }
