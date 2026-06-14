// Copyright 2026 Rahmad Afandi. MIT License.

package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/retry"
)

func TestSucceedsFirstTry(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestSucceedsAfterFailures(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	}, retry.WithAttempts(5), retry.WithDelay(time.Millisecond))
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestExhaustsAttempts(t *testing.T) {
	calls := 0
	sentinel := errors.New("always")
	err := retry.Do(context.Background(), func() error {
		calls++
		return sentinel
	}, retry.WithAttempts(4), retry.WithDelay(time.Millisecond))
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, 4, calls)
}

func TestRetryIfStopsEarly(t *testing.T) {
	calls := 0
	permanent := errors.New("permanent")
	err := retry.Do(context.Background(), func() error {
		calls++
		return permanent
	}, retry.WithAttempts(5), retry.WithDelay(time.Millisecond),
		retry.WithRetryIf(func(error) bool { return false }))
	assert.ErrorIs(t, err, permanent)
	assert.Equal(t, 1, calls, "must not retry when pred is false")
}

func TestContextCancelStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := retry.Do(ctx, func() error {
		calls++
		cancel() // cancel during the first attempt
		return errors.New("fail")
	}, retry.WithAttempts(5), retry.WithDelay(time.Hour))
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}

func TestDoValue(t *testing.T) {
	calls := 0
	v, err := retry.DoValue(context.Background(), func() (string, error) {
		calls++
		if calls < 2 {
			return "", errors.New("transient")
		}
		return "ok", nil
	}, retry.WithAttempts(3), retry.WithDelay(time.Millisecond))
	require.NoError(t, err)
	assert.Equal(t, "ok", v)
}

func TestDoValueError(t *testing.T) {
	v, err := retry.DoValue(context.Background(), func() (int, error) {
		return 0, errors.New("nope")
	}, retry.WithAttempts(2), retry.WithDelay(time.Millisecond))
	assert.Error(t, err)
	assert.Equal(t, 0, v)
}
