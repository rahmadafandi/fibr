// Copyright 2026 Rahmad Afandi. MIT License.

package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

func TestEnqueueOptions(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	c := NewClient(asynq.RedisClientOpt{Addr: mr.Addr()})
	defer c.Close()

	info, err := c.Enqueue(context.Background(), "email:send", map[string]string{"to": "a@b.c"},
		WithRetry(7), WithPriority("critical"), WithProcessIn(time.Minute))
	require.NoError(t, err)
	require.Equal(t, "critical", info.Queue)
	require.Equal(t, 7, info.MaxRetry)
	require.False(t, info.NextProcessAt.IsZero()) // ProcessIn schedules a future run
}
