// Copyright 2026 Rahmad Afandi. MIT License.

package webhook_test

import (
	"testing"
	"time"

	"github.com/rahmadafandi/fibr/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	payload := []byte(`{"event":"ok"}`)
	header := webhook.Sign(payload, "secret", time.Now())
	require.NoError(t, webhook.Verify(payload, header, "secret", webhook.DefaultTolerance))
}

func TestVerifyTamperedPayload(t *testing.T) {
	header := webhook.Sign([]byte("original"), "secret", time.Now())
	err := webhook.Verify([]byte("tampered"), header, "secret", webhook.DefaultTolerance)
	assert.ErrorIs(t, err, webhook.ErrInvalidSignature)
}

func TestVerifyWrongSecret(t *testing.T) {
	payload := []byte("body")
	header := webhook.Sign(payload, "secret", time.Now())
	err := webhook.Verify(payload, header, "other", webhook.DefaultTolerance)
	assert.ErrorIs(t, err, webhook.ErrInvalidSignature)
}

func TestVerifyExpired(t *testing.T) {
	payload := []byte("body")
	header := webhook.Sign(payload, "secret", time.Now().Add(-10*time.Minute))
	err := webhook.Verify(payload, header, "secret", 5*time.Minute)
	assert.ErrorIs(t, err, webhook.ErrExpired)
}

func TestVerifyToleranceZeroSkipsTimeCheck(t *testing.T) {
	payload := []byte("body")
	header := webhook.Sign(payload, "secret", time.Now().Add(-24*time.Hour))
	require.NoError(t, webhook.Verify(payload, header, "secret", 0))
}

func TestVerifyMalformedHeader(t *testing.T) {
	for _, h := range []string{"garbage", "t=123", "v1=abc", "t=notanint,v1=abc"} {
		err := webhook.Verify([]byte("body"), h, "secret", webhook.DefaultTolerance)
		assert.ErrorIs(t, err, webhook.ErrMalformedHeader, "header %q", h)
	}
}
