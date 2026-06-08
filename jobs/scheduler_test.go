// Copyright 2026 Rahmad Afandi. MIT License.

package jobs_test

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/rahmadafandi/fibr/jobs"
	"github.com/stretchr/testify/require"
)

func newSchedulerForTest(t *testing.T) *jobs.Scheduler {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	opt, err := jobs.RedisConnOpt("redis://" + mr.Addr())
	require.NoError(t, err)
	return jobs.NewScheduler(opt)
}

func TestSchedulerRegister(t *testing.T) {
	sched := newSchedulerForTest(t)
	id, err := sched.Register("@every 1h", "cleanup:run", map[string]int{"older_than_days": 30})
	require.NoError(t, err)
	require.NotEmpty(t, id)
}

func TestSchedulerRegisterInvalidCron(t *testing.T) {
	sched := newSchedulerForTest(t)
	_, err := sched.Register("not a cron", "cleanup:run", nil)
	require.Error(t, err)
}

func TestSchedulerRegisterBadPayload(t *testing.T) {
	sched := newSchedulerForTest(t)
	_, err := sched.Register("@every 1h", "cleanup:run", make(chan int))
	require.Error(t, err)
}

func TestWithLocation(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	opt, err := jobs.RedisConnOpt("redis://" + mr.Addr())
	require.NoError(t, err)

	// Pass a real (non-nil) location so the option path is actually exercised.
	sched := jobs.NewScheduler(opt, jobs.WithLocation(time.UTC))
	id, err := sched.Register("@every 1h", "cleanup:run", nil)
	require.NoError(t, err)
	require.NotEmpty(t, id)
}

func TestSchedulerUnregister(t *testing.T) {
	sched := newSchedulerForTest(t)
	id, err := sched.Register("@every 1h", "cleanup:run", nil)
	require.NoError(t, err)

	require.NoError(t, sched.Unregister(id))
	require.Error(t, sched.Unregister("no-such-id"))
}
