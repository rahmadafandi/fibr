// Copyright 2026 Rahmad Afandi. MIT License.

package jobs_test

import (
	"fmt"

	"github.com/alicebob/miniredis/v2"
	"github.com/rahmadafandi/fibr/jobs"
)

// Register schedules a task on a cron spec. At each tick the payload is enqueued
// as the given task type and processed by a worker's Handle. The returned entry
// id is a random uuid, so this example asserts success rather than the id.
func ExampleScheduler_Register() {
	// miniredis stands in for a real Redis here.
	mr, _ := miniredis.Run()
	defer mr.Close()

	opt, _ := jobs.RedisConnOpt("redis://" + mr.Addr())
	sched := jobs.NewScheduler(opt)

	type cleanupPayload struct {
		OlderThanDays int `json:"older_than_days"`
	}
	id, err := sched.Register("0 2 * * *", "cleanup:run", cleanupPayload{OlderThanDays: 30})

	fmt.Println("registered:", err == nil && id != "")
	// Output:
	// registered: true
}
