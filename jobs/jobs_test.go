// Copyright 2026 Rahmad Afandi. MIT License.

package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type welcomeT struct {
	Email string `json:"email"`
}

func TestRedisConnOptParses(t *testing.T) {
	_, err := RedisConnOpt("redis://localhost:6379/0")
	require.NoError(t, err)
}

func TestRedisConnOptInvalid(t *testing.T) {
	_, err := RedisConnOpt("://nope")
	assert.Error(t, err)
}

func TestHandleDecodesTypedPayload(t *testing.T) {
	srv := NewServer(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"}, ServerConfig{})
	var got string
	Handle[welcomeT](srv, "welcome:send", func(ctx context.Context, p welcomeT) error {
		got = p.Email
		return nil
	})
	payload, err := json.Marshal(welcomeT{Email: "x@y.com"})
	require.NoError(t, err)
	err = srv.ProcessTask(context.Background(), asynq.NewTask("welcome:send", payload))
	require.NoError(t, err)
	assert.Equal(t, "x@y.com", got)
}

func TestHandleBadPayloadSkipsRetry(t *testing.T) {
	srv := NewServer(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"}, ServerConfig{})
	Handle[welcomeT](srv, "welcome:send", func(ctx context.Context, p welcomeT) error { return nil })
	err := srv.ProcessTask(context.Background(), asynq.NewTask("welcome:send", []byte("not json")))
	require.Error(t, err)
	assert.True(t, errors.Is(err, asynq.SkipRetry))
}

func TestMonitoringHandlerNonNil(t *testing.T) {
	h := MonitoringHandler(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"}, "/monitoring")
	assert.NotNil(t, h)
}
