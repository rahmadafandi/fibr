// Copyright 2026 Rahmad Afandi. MIT License.

package webhook_test

import (
	"testing"
	"time"

	"github.com/rahmadafandi/fibr/webhook"
)

// FuzzVerify ensures Verify never panics on arbitrary signature headers and
// that a round-tripped signature always verifies.
func FuzzVerify(f *testing.F) {
	f.Add("t=1700000000,v1=deadbeef", []byte("payload"), "secret")
	f.Add("", []byte(""), "")
	f.Add("garbage", []byte("x"), "s")
	f.Add("t=abc,v1=", []byte("{}"), "k")

	f.Fuzz(func(t *testing.T, header string, payload []byte, secret string) {
		// Must not panic; error is fine for malformed input.
		_ = webhook.Verify(payload, header, secret, webhook.DefaultTolerance)

		// A freshly signed payload must verify (invariant).
		if secret != "" {
			sig := webhook.Sign(payload, secret, time.Now())
			if err := webhook.Verify(payload, sig, secret, webhook.DefaultTolerance); err != nil {
				t.Fatalf("round-trip verify failed: %v (sig=%q)", err, sig)
			}
		}
	})
}
